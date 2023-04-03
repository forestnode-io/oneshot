package signallers

import (
	"context"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
)

// ServerSignaller is an interface that allows a client to connect to a server.
// When a client wants to connect, the session signaller will call on the RequestHandler.
// The session signaller handles the exchange of SDP offers and answers via the AnswerOffer func it
// provides to the RequestHandler.
type ServerSignaller interface {
	// Start starts the Signaller and blocks until it is shutdown.
	Start(context.Context, RequestHandler, chan<- string) error
	// Shutdown stops the Signaller from accepting new requests.
	Shutdown() error
}

type RequestHandler interface {
	HandleRequest(context.Context, string, *webrtc.Configuration, AnswerOffer) error
}

type AnswerOffer func(context.Context, string, sdp.Offer) (sdp.Answer, error)

// HandleRequest is a function that handles a request from a client.
// A HandleRequest func is called when a client wants to connect to connect to oneshot.
// The HandleRequest func is expected to create a peer and use it create an offer to the client.
// The sdp exchange is transacted via the AnswerOffer arg.
type HandleRequest func(context.Context, string, *webrtc.Configuration, AnswerOffer) error

func (h HandleRequest) HandleRequest(ctx context.Context, id string, conf *webrtc.Configuration, offer AnswerOffer) error {
	return h(ctx, id, conf, offer)
}

type ClientSignaller interface {
	Start(context.Context, OfferHandler) error
	Shutdown() error
}

type OfferHandler interface {
	HandleOffer(context.Context, string, sdp.Offer) (sdp.Answer, error)
}
