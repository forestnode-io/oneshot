package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp/signallers"
)

// Server satisfies the sdp.RequestHandler interface.
// Server acts as a factory for new peer connections when a client request comes in.
type Server struct {
	handler http.HandlerFunc
	config  *webrtc.Configuration
	wg      sync.WaitGroup
}

func NewServer(config *webrtc.Configuration, handler http.HandlerFunc) *Server {
	return &Server{
		handler: handler,
		config:  config,
		wg:      sync.WaitGroup{},
	}
}

func (s *Server) Wait() {
	s.wg.Wait()
}

func (s *Server) HandleRequest(ctx context.Context, id string, answerOfferFunc signallers.AnswerOffer) error {
	s.wg.Add(1)
	defer s.wg.Done()

	// create a new peer connection.
	// newPeerConnection does not wait for the peer connection to be established.
	pc, pcErrs := newPeerConnection(ctx, id, answerOfferFunc, s.config)
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
			log.Println("handling http over webrtc request")
			s.handler(w, e.request)
			if w.triggersShutdown {
				done = true
			}

			log.Println("finished handling http over webrtc request")
			if err = w.Flush(); err != nil {
				return fmt.Errorf("unable to flush response: %w", err)
			}
			log.Println("flushed response to http over wenrtc request")
		}
	}

	return nil
}
