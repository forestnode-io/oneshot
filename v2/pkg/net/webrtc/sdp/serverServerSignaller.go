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

	s.t = transport.NewTransport(conn)
	log.Printf("connected to signalling server at %s", s.ssURL)

	// exchange handshake
	h := messages.Handshake{
		ID: s.id,
		VersionInfo: messages.VersionInfo{
			Version: "0.0.1",
		},
	}
	if err = s.t.Write(&h); err != nil {
		return err
	}

	m, err := s.t.Read()
	if err != nil {
		log.Printf("error reading version info: %v", err)
		return err
	}
	rh, ok := m.(*messages.Handshake)
	if !ok {
		return messages.ErrInvalidRequestType
	}

	log.Printf("signalling server version: %s", rh.VersionInfo.Version)

	// send the arrival request
	ar := messages.ArrivalRequest{
		ID:        "",
		BasicAuth: nil,
		URL:       nil,
	}
	if err = s.t.Write(&ar); err != nil {
		return err
	}

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

	log.Printf("signalling server assigned url: %s", resp.AssignedURL)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// wait for the offer request
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

			// get the offer
			if err := handler.HandleRequest(ctx, req.SessionID, s.answerOffer); err != nil {
				log.Printf("error handling request: %s", err)
				return
			}

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
	gor := messages.GetOfferResponse{
		Offer: string(offer),
	}
	if err := s.t.Write(&gor); err != nil {
		return "", err
	}

	// wait for the answer to come back
	m, err := s.t.Read()
	log.Printf("got answer: %v", m)
	if err != nil {
		log.Printf("error reading answer: %v", err)
		return "", err
	}
	a, ok := m.(*messages.GotAnswerRequest)
	log.Printf("got answer1: %v", a)
	if !ok {
		return "", messages.ErrInvalidRequestType
	}

	// let the signalling server know we got the answer
	log.Println("sending got answer response")
	gar := messages.GotAnswerResponse{}
	if err := s.t.Write(&gar); err != nil {
		return "", err
	}
	log.Println("sent got answer response")

	log.Printf("got answer2: %v", a.Answer)

	return Answer(a.Answer), nil
}
