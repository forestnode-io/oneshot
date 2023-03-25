package signallers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/proto"
	"google.golang.org/grpc"
)

type serverServerSignaller struct {
	cancel func()
	stream proto.SignallingServer_ConnectClient

	offerChan chan string

	conf *ServerServerSignallerConfig

	msgChan chan messages.Message
	errChan chan error
}

type ServerServerSignallerConfig struct {
	ID                  string
	URL                 string
	URLRequired         bool
	BasicAuth           *messages.BasicAuth
	SignallingServerURL string
	GRPCOpts            []grpc.DialOption
}

func NewServerServerSignaller(c *ServerServerSignallerConfig) ServerSignaller {
	return &serverServerSignaller{
		conf:      c,
		offerChan: make(chan string),
		msgChan:   make(chan messages.Message, 1),
		errChan:   make(chan error, 1),
	}
}

func (s *serverServerSignaller) Start(ctx context.Context, handler RequestHandler) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	grpcOpts := []grpc.DialOption{}
	if s.conf.GRPCOpts != nil {
		grpcOpts = append(grpcOpts, s.conf.GRPCOpts...)
	}
	conn, err := grpc.DialContext(ctx, s.conf.SignallingServerURL, grpcOpts...)
	if err != nil {
		return fmt.Errorf("error dialing signalling server: %w", err)
	}
	ssClient := proto.NewSignallingServerClient(conn)

	stream, err := ssClient.Connect(ctx)
	if err != nil {
		return fmt.Errorf("error connecting to signalling server: %w", err)
	}
	s.stream = stream
	go func() {
		<-ctx.Done()
		log.Println("closing connection to signalling server ...")
		if err := stream.CloseSend(); err != nil {
			log.Printf("error closing signalling server stream: %v", err)
		}
		if err := conn.Close(); err != nil {
			log.Printf("error closing signalling server connection: %v", err)
		}
	}()

	// exchange handshake
	log.Println("exchanging handshake with signalling server ...")
	h := messages.Handshake{
		ID: s.conf.ID,
		VersionInfo: messages.VersionInfo{
			Version: "0.0.1",
		},
	}
	env, err := messages.ToRPCEnvelope(&h)
	if err != nil {
		return fmt.Errorf("error marshalling handshake: %w", err)
	}
	if err = stream.Send(env); err != nil {
		return fmt.Errorf("error sending handshake: %w", err)
	}
	log.Println("... sent handshake to the signalling server...")

	log.Println("... waiting for handshake from the signalling server...")
	env, err = stream.Recv()
	if err != nil {
		return fmt.Errorf("error receiving handshake: %w", err)
	}
	m, err := messages.FromRPCEnvelope(env)
	if err != nil {
		log.Printf("error reading handshake: %v", err)
		return err
	}
	rh, ok := m.(*messages.Handshake)
	if !ok {
		return messages.ErrInvalidRequestType
	}
	log.Println("... received handshake from the signalling server...")
	log.Printf("... finished handshake with signalling server")
	log.Printf("signalling server version: %s", rh.VersionInfo.Version)
	log.Printf("signalling server api version: %s", rh.VersionInfo.APIVersion)
	if rh.ID != "" {
		log.Printf("signalling server id: %s", rh.ID)
	}

	// send the arrival request
	log.Println("sending arrival request to signalling server ...")
	ar := messages.ServerArrivalRequest{
		ID:        s.conf.ID,
		BasicAuth: s.conf.BasicAuth,
	}
	if s.conf.URL != "" {
		ar.URL = &messages.SessionURLRequest{
			URL:      s.conf.URL,
			Required: s.conf.URLRequired,
		}
	}
	env, err = messages.ToRPCEnvelope(&ar)
	if err != nil {
		return fmt.Errorf("error marshalling arrival request: %w", err)
	}
	if err = stream.Send(env); err != nil {
		return fmt.Errorf("error sending arrival request: %w", err)
	}
	log.Println("... sent arrival request to signalling server...")

	// wait for the arrival response
	env, err = stream.Recv()
	if err != nil {
		return fmt.Errorf("error receiving arrival response: %w", err)
	}
	m, err = messages.FromRPCEnvelope(env)
	if err != nil {
		return err
	}
	resp, ok := m.(*messages.ServerArrivalResponse)
	if !ok {
		return messages.ErrInvalidRequestType
	}
	if resp.Error != "" {
		return fmt.Errorf("signalling server responded with error: %s", resp.Error)
	}
	log.Println("... received arrival response from signalling server.")

	if resp.AssignedURL != "" {
		log.Printf("signalling server assigned url: %s", resp.AssignedURL)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("context cancelled, closing conenction with the signalling server")
			return nil
		default:
		}

		// wait for the offer request
		log.Println("waiting for offer request from signalling server ...")
		env, err = stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Println("signalling server closed connection")
				return nil
			}
			return fmt.Errorf("error receiving offer request: %w", err)
		}
		m, err = messages.FromRPCEnvelope(env)
		if err != nil {
			return fmt.Errorf("error reading offer request: %w", err)
		}

		switch m := m.(type) {
		case *messages.GetOfferRequest:
			log.Println("... got offer request from signalling server")

			// get the offer
			log.Println("sending offer request to handler ...")
			if err := handler.HandleRequest(ctx, m.SessionID, m.Configuration, s.answerOffer); err != nil {
				return fmt.Errorf("error handling request: %w", err)
			}
			log.Println("... handler finished processing offer request")
		default:
			return fmt.Errorf("unexpected message type: %T", m)
		}
	}
}

func (s *serverServerSignaller) Shutdown() error {
	s.cancel()
	return nil
}

func (s *serverServerSignaller) answerOffer(ctx context.Context, sessionID string, offer sdp.Offer) (sdp.Answer, error) {
	// send the offer to the signalling server
	log.Println("sending offer to signalling server ...")
	gor := messages.GetOfferResponse{
		Offer: string(offer),
	}
	env, err := messages.ToRPCEnvelope(&gor)
	if err != nil {
		return "", fmt.Errorf("error marshalling offer: %w", err)
	}
	if err = s.stream.Send(env); err != nil {
		return "", fmt.Errorf("error sending offer: %w", err)
	}
	log.Println("... sent offer to signalling server")

	// wait for the answer to come back
	log.Println("waiting for answer from signalling server ...")
	env, err = s.stream.Recv()
	if err != nil {
		return "", fmt.Errorf("error receiving answer: %w", err)
	}
	m, err := messages.FromRPCEnvelope(env)
	if err != nil {
		log.Printf("error reading answer: %v", err)
		return "", err
	}

	a, ok := m.(*messages.GotAnswerRequest)
	if !ok {
		return "", messages.ErrInvalidRequestType
	}

	log.Println("got answer from signalling server")
	log.Println("verifying with signalling server that answer was received ...")

	// let the signalling server know we got the answer
	gar := messages.GotAnswerResponse{}
	env, err = messages.ToRPCEnvelope(&gar)
	if err != nil {
		return "", fmt.Errorf("error marshalling answer received: %w", err)
	}
	if err = s.stream.Send(env); err != nil {
		return "", fmt.Errorf("error sending answer received: %w", err)
	}

	log.Println("... verified to signalling server that answer was received")

	return sdp.Answer(a.Answer), nil
}
