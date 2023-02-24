package webrtc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
)

const dataChannelName = "oneshot"

const (
	dataChannelMTU             = 16384              // 16 KB
	bufferedAmountLowThreshold = 1 * dataChannelMTU // 2^0 MTU
	maxBufferedAmount          = 8 * dataChannelMTU // 2^3 MTUs
)

// Subsystem satisfies the sdp.RequestHandler interface.
// Subsystem acts as a factory for new peer connections when a client request comes in.
type Subsystem struct {
	Handler http.HandlerFunc
	Config  *webrtc.Configuration
}

func (s *Subsystem) HandleRequest(ctx context.Context, answerOfferFunc sdp.AnswerOffer) error {
	// create a new peer connection.
	// newPeerConnection does not wait for the peer connection to be established.
	pc, pcErrs := newPeerConnection(ctx, answerOfferFunc, s.Config)
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
			return ctx.Err()
		case e := <-pcErrs:
			return fmt.Errorf("error on peer connection: %w", e)
		case e := <-d.eventsChan:
			if e.err != nil {
				return fmt.Errorf("error on data channel: %w", e.err)
			}

			w := newHTTPResponseWriter(d)
			s.Handler(w, e.request)
			// the httpResponseWriter will send the header as a string and the body as a byte slice.
			// when sending http over webrtc we signal the end of the response by sending an EOF as a string.
			w.channel.WriteDataChannel([]byte("EOF"), true)
		}
	}
}
