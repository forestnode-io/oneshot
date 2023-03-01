package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
)

// Server satisfies the sdp.RequestHandler interface.
// Server acts as a factory for new peer connections when a client request comes in.
type Server struct {
	Handler http.HandlerFunc
	Config  *webrtc.Configuration
}

func (s *Server) HandleRequest(ctx context.Context, id int32, answerOfferFunc sdp.AnswerOffer) error {
	// create a new peer connection.
	// newPeerConnection does not wait for the peer connection to be established.
	pc, pcErrs := newPeerConnection(ctx, id, answerOfferFunc, s.Config)
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

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("webrtc server context cancelled: %w", ctx.Err())
		case e := <-pcErrs:
			return fmt.Errorf("error on peer connection: %w", e)
		case e := <-d.eventsChan:
			if e.err != nil {
				return fmt.Errorf("error on data channel: %w", e.err)
			}

			w := NewResponseWriter(d)
			s.Handler(w, e.request)
			if err = w.Flush(); err != nil {
				return fmt.Errorf("unable to flush response: %w", err)
			}
		}
	}
}
