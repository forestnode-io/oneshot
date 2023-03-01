package sdp

import (
	"context"
	"fmt"
	"log"
	"net"

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
}

func NewServerServerSignaller(ssURL, id, url string, urlRequired bool) ServerSignaller {
	return &serverServerSignaller{
		ssURL:       ssURL,
		id:          id,
		url:         url,
		urlRequired: urlRequired,
		offerChan:   make(chan string),
	}
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

	s.t = transport.NewTransport(conn)

	// exchange handshake
	log.Println("exchanging handshake with signalling server ...")
	h := messages.Handshake{
		ID: s.id,
		VersionInfo: messages.VersionInfo{
			Version: "0.0.1",
		},
	}
	if err = s.t.Write(&h); err != nil {
		return err
	}
	log.Println("... sent handshake to the signalling server...")

	log.Println("... waiting for handshake from the signalling server...")
	m, err := s.t.Read()
	if err != nil {
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
	ar := messages.ArrivalRequest{
		ID:        "",
		BasicAuth: nil,
		URL:       nil,
	}
	if err = s.t.Write(&ar); err != nil {
		return err
	}
	log.Println("... sent arrival request to signalling server...")

	// wait for the arrival response
	m, err = s.t.Read()
	if err != nil {
		return err
	}
	resp, ok := m.(*messages.ArrivalResponse)
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

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// wait for the offer request
			log.Println("waiting for offer request from signalling server ...")
			m, err := s.t.Read()
			if err != nil {
				log.Printf("error reading from signalling server: %s", err)
				return
			}
			req, ok := m.(*messages.GetOfferRequest)
			if !ok {
				log.Printf("invalid request type from signalling server: %T", m)
				return
			}
			log.Println("... got offer request from signalling server")

			// get the offer
			log.Println("sending offer request to handler ...")
			if err := handler.HandleRequest(ctx, req.SessionID, s.answerOffer); err != nil {
				log.Printf("error handling request: %s", err)
				return
			}
			log.Println("... handler finished processing offer request")
		}
	}()

	return nil
}

func (s *serverServerSignaller) Shutdown() error {
	s.cancel()
	return nil
}

func (s *serverServerSignaller) answerOffer(ctx context.Context, sessionID int32, offer Offer) (Answer, error) {
	// send the offer to the signalling server
	log.Println("sending offer to signalling server ...")
	gor := messages.GetOfferResponse{
		Offer: string(offer),
	}
	if err := s.t.Write(&gor); err != nil {
		return "", err
	}
	log.Println("... sent offer to signalling server")

	// wait for the answer to come back
	log.Println("waiting for answer from signalling server ...")
	m, err := s.t.Read()
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
	if err := s.t.Write(&gar); err != nil {
		return "", err
	}

	log.Println("... verified to signalling server that answer was received")

	return Answer(a.Answer), nil
}
