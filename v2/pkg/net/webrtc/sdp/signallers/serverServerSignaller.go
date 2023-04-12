package signallers

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/rs/zerolog"
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
	log := zerolog.Ctx(ctx)

	ds := signallingserver.GetDiscoveryServer(ctx)
	if ds == nil {
		return errors.New("discovery server not found")
	}
	s.ds = ds

	go func() {
		<-ctx.Done()
		log.Debug().Msg("closing connection to discovery server")
		if err := ds.Close(); err != nil {
			log.Error().Err(err).
				Msg("error closing connection to discovery server")
		}
	}()

	// send the arrival request
	log.Debug().Msg("sending arrival request to discovery server")
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
	log.Debug().
		Msg("sent arrival request to discovery server, waiting for arrival response from discovery server")

	// wait for the arrival response
	sap, err := signallingserver.Receive[*messages.ServerArrivalResponse](ds)
	if err != nil {
		return fmt.Errorf("error receiving arrival response: %w", err)
	}
	if sap.Error != "" {
		return fmt.Errorf("discovery server responded with error: %s", sap.Error)
	}
	log.Debug().
		Msg("received arrival response from discovery server, waiting for discovery server to assign url")

	if sap.AssignedURL == "" {
		return fmt.Errorf("discovery server did not assign a url")
	}
	log.Printf("discovery server assigned url: %s", sap.AssignedURL)
	addressChan <- sap.AssignedURL
	close(addressChan)

	for {
		select {
		case <-ctx.Done():
			log.Debug().
				Msg("context done, closing connection to discovery server")
			return nil
		default:
		}

		// wait for the offer request
		log.Debug().
			Msg("waiting for offer request from discovery server")
		gor, err := signallingserver.Receive[*messages.GetOfferRequest](ds)
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Debug().
					Msg("discovery server closed connection")
				return nil
			}
			return fmt.Errorf("error receiving offer request: %w", err)
		}

		log.Debug().
			Msg("got offer request from discovery server")

		// get the offer
		if err := handler.HandleRequest(ctx, gor.SessionID, gor.Configuration, s.answerOffer); err != nil {
			log.Error().Err(err).
				Msg("error handling offer request")

			err = signallingserver.Send(ds, &messages.FinishedSessionRequest{
				SessionID: gor.SessionID,
				Error:     err.Error(),
			})
			if err != nil {
				return fmt.Errorf("error sending session failed request: %w", err)
			}
			continue
		}
		log.Debug().
			Msg("handler finished processing offer request")
	}
}

func (s *serverServerSignaller) Shutdown() error {
	s.cancel()
	return nil
}

func (s *serverServerSignaller) answerOffer(ctx context.Context, sessionID string, offer sdp.Offer) (sdp.Answer, error) {
	log := zerolog.Ctx(ctx)
	// send the offer to the signalling server
	log.Debug().
		Msg("sending offer request to discovery server")
	err := signallingserver.Send(s.ds, &messages.GetOfferResponse{
		Offer: string(offer),
	})
	if err != nil {
		return "", fmt.Errorf("error sending offer: %w", err)
	}
	log.Debug().
		Msg("sent offer to discovery server, waiting for answer from discovery server")

	// wait for the answer to come back
	gar, err := signallingserver.Receive[*messages.GotAnswerRequest](s.ds)
	if err != nil {
		return "", fmt.Errorf("error receiving answer: %w", err)
	}

	log.Debug().
		Msg("go answer from discovery server")

	// let the signalling server know we got the answer
	err = signallingserver.Send(s.ds, &messages.GotAnswerResponse{})
	if err != nil {
		return "", fmt.Errorf("error sending answer received: %w", err)
	}

	log.Debug().
		Msg("sent answer received to discovery server")

	return sdp.Answer(gar.Answer), nil
}
