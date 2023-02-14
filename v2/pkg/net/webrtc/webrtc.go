package webrtc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
)

// Subsystem satisfies the sdp.RequestHandler interface.
// Subsystem acts as a factory for new peer connections when a client request comes in.
type Subsystem struct {
	Handler http.HandlerFunc
	Config  *webrtc.Configuration
}

func (s *Subsystem) HandleRequest(ctx context.Context, offer sdp.AnswerOffer) error {
	pc, pcErrs := newPeerConnection(ctx, offer, s.Config)
	if pc == nil {
		err := <-pcErrs
		err = fmt.Errorf("unable to create new webRTC peer connection: %w", err)
		return err
	}
	defer pc.Close()

	dc, dcErrs := newDataChannel("", pc, s.Handler)
	if dc == nil {
		err := <-dcErrs
		err = fmt.Errorf("unable to create data channel for webRTC peer connection: %w", err)
		return err
	}
	defer dc.Close()

	// TODO(raphaelreyna): handle errors from errs / block until done
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case e := <-pcErrs:
		err = e
	case e := <-dcErrs:
		err = e
	}

	return err
}
