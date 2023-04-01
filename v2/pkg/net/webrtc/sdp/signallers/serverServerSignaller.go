package signallers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
)

type serverServerSignaller struct {
	cancel func()

	ds *signallingserver.DiscoveryServer

	offerChan chan string

	conf *ServerServerSignallerConfig

	msgChan chan messages.Message
	errChan chan error
}

type ServerServerSignallerConfig struct {
	URL         string
	URLRequired bool
	BasicAuth   *messages.BasicAuth
	PortMapAddr string
}

func NewServerServerSignaller(c *ServerServerSignallerConfig) ServerSignaller {
	return &serverServerSignaller{
		conf:      c,
		offerChan: make(chan string),
		msgChan:   make(chan messages.Message, 1),
		errChan:   make(chan error, 1),
	}
}

func (s *serverServerSignaller) Start(ctx context.Context, handler RequestHandler, addressChan chan<- string) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	ds := signallingserver.GetDiscoveryServer(ctx)
	if ds == nil {
		return errors.New("discovery server not found")
	}
	s.ds = ds

	go func() {
		<-ctx.Done()
		log.Println("closing connection to signalling server ...")
		if err := ds.Close(); err != nil {
			log.Printf("error closing connection to signalling server: %v", err)
		}
	}()

	// send the arrival request
	log.Println("sending arrival request to signalling server ...")
	ar := messages.ServerArrivalRequest{
		BasicAuth: s.conf.BasicAuth,
		Redirect:  s.conf.PortMapAddr,
	}
	if s.conf.URL != "" {
		ar.URL = &messages.SessionURLRequest{
			URL:      s.conf.URL,
			Required: s.conf.URLRequired,
		}
	}
	err := signallingserver.Send(ds, &ar)
	if err != nil {
		return fmt.Errorf("error marshalling arrival request: %w", err)
	}
	log.Println("... sent arrival request to signalling server...")

	// wait for the arrival response
	sap, err := signallingserver.Receive[*messages.ServerArrivalResponse](ds)
	if err != nil {
		return fmt.Errorf("error receiving arrival response: %w", err)
	}
	if sap.Error != "" {
		return fmt.Errorf("signalling server responded with error: %s", sap.Error)
	}
	log.Println("... received arrival response from signalling server.")

	if sap.AssignedURL == "" {
		return fmt.Errorf("signalling server did not assign a url")
	}
	log.Printf("signalling server assigned url: %s", sap.AssignedURL)
	addressChan <- sap.AssignedURL
	close(addressChan)

	for {
		select {
		case <-ctx.Done():
			log.Println("context cancelled, closing conenction with the signalling server")
			return nil
		default:
		}

		// wait for the offer request
		log.Println("waiting for offer request from signalling server ...")
		gor, err := signallingserver.Receive[*messages.GetOfferRequest](ds)
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Println("signalling server closed connection")
				return nil
			}
			return fmt.Errorf("error receiving offer request: %w", err)
		}

		log.Println("... got offer request from signalling server")

		// get the offer
		if err := handler.HandleRequest(ctx, gor.SessionID, gor.Configuration, s.answerOffer); err != nil {
			err = signallingserver.Send(ds, &messages.FinishedSessionRequest{
				SessionID: gor.SessionID,
				Error:     err.Error(),
			})
			if err != nil {
				return fmt.Errorf("error sending session failed request: %w", err)
			}
			log.Printf("error handling offer request: %v", err)
			continue
		}
		log.Println("... handler finished processing offer request")
	}
}

func (s *serverServerSignaller) Shutdown() error {
	s.cancel()
	return nil
}

func (s *serverServerSignaller) answerOffer(ctx context.Context, sessionID string, offer sdp.Offer) (sdp.Answer, error) {
	// send the offer to the signalling server
	log.Println("sending offer to signalling server ...")
	err := signallingserver.Send(s.ds, &messages.GetOfferResponse{
		Offer: string(offer),
	})
	if err != nil {
		return "", fmt.Errorf("error sending offer: %w", err)
	}
	log.Println("... sent offer to signalling server")

	// wait for the answer to come back
	gar, err := signallingserver.Receive[*messages.GotAnswerRequest](s.ds)
	if err != nil {
		return "", fmt.Errorf("error receiving answer: %w", err)
	}

	log.Println("got answer from signalling server")
	log.Println("verifying with signalling server that answer was received ...")

	// let the signalling server know we got the answer
	err = signallingserver.Send(s.ds, &messages.GotAnswerResponse{})
	if err != nil {
		return "", fmt.Errorf("error sending answer received: %w", err)
	}

	log.Println("... verified to signalling server that answer was received")

	return sdp.Answer(gar.Answer), nil
}
