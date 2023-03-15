package server

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp/signallers"
)

type peerConnection struct {
	ctx         context.Context
	errChan     chan<- error
	answerOffer signallers.AnswerOffer
	sessionID   string

	peerAddresses []string
	paMu          sync.Mutex

	*webrtc.PeerConnection
}

func newPeerConnection(ctx context.Context, id string, sao signallers.AnswerOffer, c *webrtc.Configuration) (*peerConnection, <-chan error) {
	var (
		err  error
		errs = make(chan error, 1)
	)

	pc := peerConnection{
		ctx:         ctx,
		errChan:     errs,
		answerOffer: sao,
		sessionID:   id,
	}

	se := webrtc.SettingEngine{}
	se.DetachDataChannels()
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	pc.PeerConnection, err = api.NewPeerConnection(*c)
	if err != nil {
		errs <- fmt.Errorf("unable to create new webRTC peer connection: %w", err)
		return nil, errs
	}

	ppc := pc.PeerConnection
	ppc.OnICEConnectionStateChange(pc.onICEConnectionStateChange)
	ppc.OnNegotiationNeeded(pc.onNegotiationNeeded)
	ppc.OnICECandidate(pc.onICECandidate)
	ppc.OnConnectionStateChange(pc.onConnectionStateChange)
	ppc.OnSignalingStateChange(pc.onSignalingStateChange)

	return &pc, errs
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
	if candidate != nil {
		log.Println("ICE candidate gathered", candidate)
		p.paMu.Lock()
		p.peerAddresses = append(p.peerAddresses, fmt.Sprintf("%s:%d", candidate.Address, candidate.Port))
		p.paMu.Unlock()
	} else {
		log.Println("ICE candidate gathering complete")
	}

	// candidate is nil when gathering is done
	if doneGathering := candidate == nil; doneGathering {
		pc := p.PeerConnection
		answer, err := p.answerOffer(p.ctx, p.sessionID, sdp.Offer(pc.LocalDescription().SDP))
		if err != nil {
			err = fmt.Errorf("unable to exchange session description with signaling server: %w", err)
			p.error(true, err)
		}

		answerSD, err := answer.WebRTCSessionDescription()
		if err != nil {
			err = fmt.Errorf("unable to convert session description to webRTC session description: %w", err)
			p.error(true, err)
		}

		if err := pc.SetRemoteDescription(answerSD); err != nil {
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
	log.Println("webrtc peer connection negotiation needed ...")
	defer log.Println("... peer connection negotiated")

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

func (p *peerConnection) getPeerAddresses() []string {
	p.paMu.Lock()
	defer p.paMu.Unlock()
	return p.peerAddresses
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
