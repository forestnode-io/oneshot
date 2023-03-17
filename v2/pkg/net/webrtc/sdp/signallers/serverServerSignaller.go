package signallers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/transport"
)

type serverServerSignaller struct {
	cancel func()
	ssURL  string
	id     string
	t      *transport.Transport

	url         string
	urlRequired bool
	offerChan   chan string

	heartbeatPeriod time.Duration
	heartbeatMu     sync.Mutex

	msgChan chan messages.Message
	errChan chan error
}

func NewServerServerSignaller(ssURL, id, url string, urlRequired bool) ServerSignaller {
	return &serverServerSignaller{
		ssURL:           ssURL,
		id:              id,
		url:             url,
		urlRequired:     urlRequired,
		offerChan:       make(chan string),
		msgChan:         make(chan messages.Message, 1),
		errChan:         make(chan error, 1),
		heartbeatPeriod: sdp.PingWindowDuration / 2,
	}
}

func (s *serverServerSignaller) startHeartbeat(ctx context.Context) {
	ping := messages.Ping{}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(s.heartbeatPeriod):
				err := s.write(&ping)
				if err != nil {
					if strings.Contains(err.Error(), "broken pipe") {
						return
					}
					log.Printf("error sending heartbeat: %v", err)
				}
			}
		}
	}()
}

func (s *serverServerSignaller) startReadPump(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				m, err := s.t.Read()
				if err != nil {
					s.errChan <- err
					close(s.msgChan)
					close(s.errChan)
					return
				}

				u, ok := m.(*messages.UpdatePingRateRequest)
				if ok {
					if u.Period != 0 {
						s.heartbeatPeriod = u.Period
						log.Println("updated heartbeat period to", s.heartbeatPeriod)
					}
					continue
				}

				s.msgChan <- m
			}
		}
	}()
}

func (s *serverServerSignaller) read() (messages.Message, error) {
	select {
	case m := <-s.msgChan:
		return m, nil
	case err := <-s.errChan:
		return nil, err
	}
}

func (s *serverServerSignaller) write(m messages.Message) error {
	s.heartbeatMu.Lock()
	defer s.heartbeatMu.Unlock()
	err := s.t.Write(m)

	return err
}

func (s *serverServerSignaller) Start(ctx context.Context, handler RequestHandler) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	log.Printf("connecting to signalling server at %s", s.ssURL)
	conn, err := net.Dial("tcp", s.ssURL)
	if err != nil {
		return fmt.Errorf("unable to connect to signalling server: %w", err)
	}
	log.Printf("connected to signalling server at %s", s.ssURL)

	go func() {
		<-ctx.Done()
		log.Println("closing connection to signalling server ...")
		conn.Close()
	}()

	s.t = transport.NewTransport(conn)
	s.startReadPump(ctx)

	// exchange handshake
	log.Println("exchanging handshake with signalling server ...")
	h := messages.Handshake{
		ID: s.id,
		VersionInfo: messages.VersionInfo{
			Version: "0.0.1",
		},
	}
	if err = s.write(&h); err != nil {
		return err
	}
	log.Println("... sent handshake to the signalling server...")

	log.Println("... waiting for handshake from the signalling server...")
	m, err := s.read()
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		log.Printf("error reading version info: %v", err)
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
		ID:        s.id,
		BasicAuth: nil,
		URL:       nil,
	}
	if err = s.write(&ar); err != nil {
		return err
	}
	log.Println("... sent arrival request to signalling server...")

	// wait for the arrival response
	m, err = s.read()
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

	s.startHeartbeat(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("context cancelled, closing conenction with the signalling server")
			return nil
		default:
		}

		// wait for the offer request
		log.Println("waiting for offer request from signalling server ...")
		m, err := s.read()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("error reading from signalling server: %w", err)
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
	if err := s.write(&gor); err != nil {
		return "", err
	}
	log.Println("... sent offer to signalling server")

	// wait for the answer to come back
	log.Println("waiting for answer from signalling server ...")
	m, err := s.read()
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
	if err := s.write(&gar); err != nil {
		return "", err
	}

	log.Println("... verified to signalling server that answer was received")

	return sdp.Answer(a.Answer), nil
}
