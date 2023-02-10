package webrtc

import (
	"context"
	"fmt"
	"log"

	"github.com/pion/webrtc/v3"
)

type configuration struct {
	ClientRequestID int
	SDExchange
	*webrtc.Configuration
}

type peerConnection struct {
	ctx             context.Context
	clientRequestID int
	sdExchange      SDExchange
	errChan         chan<- error

	*webrtc.PeerConnection
}

func newPeerConnection(ctx context.Context, ec chan error, c configuration) (*peerConnection, error) {
	var err error
	pc := peerConnection{
		ctx:             ctx,
		clientRequestID: c.ClientRequestID,
		errChan:         ec,
		sdExchange:      c.SDExchange,
	}

	pc.PeerConnection, err = webrtc.NewPeerConnection(*c.Configuration)
	if err != nil {
		return nil, fmt.Errorf("unable to create new webRTC peer connection: %w", err)
	}

	ppc := pc.PeerConnection
	ppc.OnICEConnectionStateChange(pc.onICEConnectionStateChange)
	ppc.OnNegotiationNeeded(pc.onNegotiationNeeded)
	ppc.OnICECandidate(pc.onICECandidate)
	ppc.OnConnectionStateChange(pc.onConnectionStateChange)
	ppc.OnSignalingStateChange(pc.onSignalingStateChange)

	return &pc, nil
}

func (p *peerConnection) onICEConnectionStateChange(cs webrtc.ICEConnectionState) {
	log.Printf("connection state has changed: %s\n", cs.String())

	switch cs {
	case webrtc.ICEConnectionStateConnected:
		log.Println("webRTC connection established")
	case webrtc.ICEConnectionStateDisconnected:
		p.error(false, fmt.Errorf("webRTC connection disconnected"))
	case webrtc.ICEConnectionStateFailed:
		p.error(false, fmt.Errorf("webRTC connection failed"))
	case webrtc.ICEConnectionStateClosed:
		p.error(false, fmt.Errorf("webRTC connection closed"))
	}
}

func (p *peerConnection) onICECandidate(candidate *webrtc.ICECandidate) {
	log.Println("ICE candidate gathered", candidate)

	// candidate is nil when gathering is done
	if doneGathering := candidate == nil; doneGathering {
		pc := p.PeerConnection
		clientSD, err := p.sdExchange(p.ctx, p.clientRequestID, pc.LocalDescription())
		if err != nil {
			err = fmt.Errorf("unable to exchange session description with signaling server: %w", err)
			p.error(true, err)
		}

		if err := pc.SetRemoteDescription(*clientSD); err != nil {
			err = fmt.Errorf("unable to set remote session description during webRTC negotiation: %w", err)
			p.error(true, err)
		}
		return
	}
}

func (p *peerConnection) onConnectionStateChange(state webrtc.PeerConnectionState) {
	log.Printf("webRTC connection state changed: %s", state.String())

	switch state {
	case webrtc.PeerConnectionStateNew:
		log.Println("webRTC connection new")
	case webrtc.PeerConnectionStateConnecting:
		log.Println("webRTC connection connecting")
	case webrtc.PeerConnectionStateConnected:
		log.Println("webRTC connection established")
	case webrtc.PeerConnectionStateDisconnected:
		p.error(false, fmt.Errorf("webRTC connection disconnected"))
	case webrtc.PeerConnectionStateFailed:
		p.error(false, fmt.Errorf("webRTC connection failed"))
	case webrtc.PeerConnectionStateClosed:
		p.error(false, fmt.Errorf("webRTC connection closed"))
	}
}

func (p *peerConnection) onSignalingStateChange(state webrtc.SignalingState) {
	log.Printf("webRTC signaling state changed: %s", state.String())

	switch state {
	case webrtc.SignalingStateStable:
		log.Println("webRTC signaling stable")
	case webrtc.SignalingStateHaveLocalOffer:
		log.Println("webRTC signaling have local offer")
	case webrtc.SignalingStateHaveRemoteOffer:
		log.Println("webRTC signaling have remote offer")
	case webrtc.SignalingStateHaveLocalPranswer:
		log.Println("webRTC signaling have local pranswer")
	case webrtc.SignalingStateHaveRemotePranswer:
		log.Println("webRTC signaling have remote pranswer")
	case webrtc.SignalingStateClosed:
		log.Println("webRTC signaling closed")
	}
}

func (p *peerConnection) onNegotiationNeeded() {
	log.Println("negotiation needed")

	pc := p.PeerConnection
	// ready to negotiate a new offer session description
	sd, err := pc.CreateOffer(nil)
	if err != nil {
		err = fmt.Errorf("unable to create offer session description during webRTC negotiation: %w", err)
		p.error(true, err)
	}

	if err := pc.SetLocalDescription(sd); err != nil {
		err = fmt.Errorf("unable to set local session description during webRTC negotiation: %w", err)
		p.error(true, err)
	}
}

func (p *peerConnection) error(local bool, err error) {
	go func() {
		e := newPeerConnectionError(err)
		e.local = local
		p.errChan <- e
	}()
}

type peerConnectionError struct {
	error
	local bool
}

func (e *peerConnectionError) Error() string {
	return e.error.Error()
}

func (e *peerConnectionError) Unwrap() error {
	return e.error
}

func newPeerConnectionError(err error) *peerConnectionError {
	return &peerConnectionError{error: err}
}
