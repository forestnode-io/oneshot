package webrtc

import (
	"bufio"
	"context"
	"fmt"
	"net/http"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
)

const dataChannelName = "oneshot"

const (
	dataChannelMTU             = 16384               // 16 KB
	bufferedAmountLowThreshold = 1 * dataChannelMTU  // 1 MTU
	maxBufferedAmount          = 10 * dataChannelMTU // 10 MTUs
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
	d, dcErrs := newDataChannel(pc)
	if d == nil {
		err := <-dcErrs
		err = fmt.Errorf("unable to create new data channel: %w", err)
		return err
	}
	defer d.Close()

	buf := bufio.NewReader(d)
	r, err := http.ReadRequest(buf)
	if err != nil {
		return fmt.Errorf("unable to read http request from data channel: %w", err)
	}
	w := newHTTPResponseWriter(d)

	s.Handler(w, r)

	w.channel.WriteDataChannel([]byte("EOF"), true)

	// TODO(raphaelreyna): handle errors from errs / block until done
	//var err error
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
