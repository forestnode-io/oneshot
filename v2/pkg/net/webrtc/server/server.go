package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/sdp/signallers"
)

// Server satisfies the sdp.RequestHandler interface.
// Server acts as a factory for new peer connections when a client request comes in.
type Server struct {
	handler http.HandlerFunc
	config  *webrtc.Configuration
	wg      sync.WaitGroup
	bat     string
}

func NewServer(config *webrtc.Configuration, bat string, handler http.HandlerFunc) *Server {
	return &Server{
		handler: handler,
		config:  config,
		wg:      sync.WaitGroup{},
		bat:     bat,
	}
}

func (s *Server) Wait() {
	s.wg.Wait()
}

func (s *Server) HandleRequest(ctx context.Context, id string, conf *webrtc.Configuration, answerOfferFunc signallers.AnswerOffer) error {
	s.wg.Add(1)
	defer s.wg.Done()

	if conf == nil {
		conf = s.config
	}
	// create a new peer connection.
	// newPeerConnection does not wait for the peer connection to be established.
	pc, pcErrs := newPeerConnection(ctx, id, s.bat, answerOfferFunc, conf)
	if pc == nil {
		err := <-pcErrs
		err = fmt.Errorf("unable to create new webRTC peer connection: %w", err)
		return err
	}
	defer pc.Close()

	// create a new data channel.
	// newDataChannel waits for the data channel to be established.
	d, err := newDataChannel(ctx, pc)
	if err != nil {
		return fmt.Errorf("unable to create new webRTC data channel: %w", err)
	}
	defer d.Close()

	var done bool
	for !done {
		select {
		case <-ctx.Done():
			return nil
		case e := <-pcErrs:
			return fmt.Errorf("error on peer connection: %w", e)
		case e := <-d.eventsChan:
			if e.err != nil {
				return fmt.Errorf("error on data channel: %w", e.err)
			}

			w := NewResponseWriter(d)
			s.handler(w, e.request)
			if w.triggersShutdown {
				done = true
			}

			if err = w.Flush(); err != nil {
				return fmt.Errorf("unable to flush response: %w", err)
			}
		}
	}

	return nil
}
