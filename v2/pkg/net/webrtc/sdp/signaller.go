package sdp

import "context"

// ServerSignaller is an interface that allows a client to connect to a server.
// When a client wants to connect, the session signaller will call on the RequestHandler.
// The session signaller handles the exchange of SDP offers and answers via the AnswerOffer func it
// provides to the RequestHandler.
type ServerSignaller interface {
	Start(context.Context, RequestHandler) error
	// Shutdown stops the Signaller from accepting new requests.
	Shutdown() error
}

type RequestHandler interface {
	HandleRequest(context.Context, AnswerOffer) error
}

type AnswerOffer func(context.Context, Offer) (Answer, error)

// HandleRequest is a function that handles a request from a client.
// A HandleRequest func is called when a client wants to connect to connect to oneshot.
// The HandleRequest func is expected to create a peer and use it create an offer to the client.
// The sdp exchange is transacted via the AnswerOffer arg.
type HandleRequest func(context.Context, AnswerOffer) error

func (h HandleRequest) HandleRequest(ctx context.Context, offer AnswerOffer) error {
	return h(ctx, offer)
}

type ClientSignaller interface {
	Start(context.Context, OfferHandler) error
	Shutdown() error
}

type OfferHandler interface {
	HandleOffer(context.Context, Offer) (Answer, error)
}
