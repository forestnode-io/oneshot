package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/oneshot-uno/oneshot/v2/pkg/log"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/sdp/signallers"
	"github.com/pion/webrtc/v3"
)

type peerConnection struct {
	ctx            context.Context
	errChan        chan<- error
	answerOffer    signallers.AnswerOffer
	answeredOffer  bool
	sessionID      string
	basicAuthToken string

	iceGatherTimeout time.Duration

	peerAddresses []string
	paMu          sync.Mutex

	cancelTimer func() bool

	*webrtc.PeerConnection
}

func newPeerConnection(ctx context.Context, id, bat string, gatherTimeout time.Duration, sao signallers.AnswerOffer, c *webrtc.Configuration) (*peerConnection, <-chan error) {
	var (
		err  error
		errs = make(chan error, 1)
	)

	pc := peerConnection{
		ctx:              ctx,
		errChan:          errs,
		answerOffer:      sao,
		sessionID:        id,
		basicAuthToken:   bat,
		iceGatherTimeout: gatherTimeout,
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
	log := log.Logger()
	log.Debug().
		Str("connection_state", cs.String()).
		Msg("ICE connection state changed")

	switch cs {
	case webrtc.ICEConnectionStateChecking:
		t := time.AfterFunc(3*time.Second, func() {
			p.error(false, fmt.Errorf("webRTC connection timed out"))
		})
		p.cancelTimer = t.Stop
	case webrtc.ICEConnectionStateConnected:
		if p.cancelTimer != nil {
			p.cancelTimer()
		}
		log.Debug().
			Msg("webRTC connection established")
	case webrtc.ICEConnectionStateDisconnected:
		p.error(false, fmt.Errorf("webRTC connection disconnected"))
	case webrtc.ICEConnectionStateFailed:
		p.error(false, fmt.Errorf("webRTC connection failed"))
	case webrtc.ICEConnectionStateClosed:
		p.error(false, fmt.Errorf("webRTC connection closed"))
	}
}

func (p *peerConnection) _answerOffer() {
	log := log.Logger()
	pc := p.PeerConnection
	offer := pc.LocalDescription()

	p.paMu.Lock()
	defer p.paMu.Unlock()

	if p.answeredOffer {
		return
	}

	if offer == nil || len(p.peerAddresses) == 0 {
		err := fmt.Errorf("unable to get local session description during webRTC negotiation")
		p.error(true, err)
		return
	}

	// if we have a basic auth token, add it to the session description
	if p.basicAuthToken != "" {
		sdp, err := offer.Unmarshal()
		if err != nil {
			err = fmt.Errorf("unable to unmarshal session description: %w", err)
			p.error(true, err)
		}
		sdp = sdp.WithValueAttribute("BasicAuthToken", p.basicAuthToken)
		sdpBytes, err := sdp.Marshal()
		if err != nil {
			err = fmt.Errorf("unable to marshal session description: %w", err)
			p.error(true, err)
			return
		}
		offer.SDP = string(sdpBytes)
	}

	log.Debug().
		Msg("sending offer to signaling server")

	answer, err := p.answerOffer(p.ctx, p.sessionID, sdp.Offer(offer.SDP))
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

	p.answeredOffer = true
}

func (p *peerConnection) onICECandidate(candidate *webrtc.ICECandidate) {
	log := log.Logger()
	if candidate != nil {
		log.Debug().
			Interface("candidate", candidate).
			Msg("ICE candidate gathered")
		p.paMu.Lock()
		p.peerAddresses = append(p.peerAddresses, fmt.Sprintf("%s:%d", candidate.Address, candidate.Port))
		p.paMu.Unlock()
	} else {
		log.Debug().
			Msg("ICE candidate gathering complete")
	}

	// candidate is nil when gathering is done
	if doneGathering := candidate == nil; doneGathering {
		log.Debug().
			Msg("done gathering ICE candidates, sending offer to signaling server")

		p._answerOffer()

		return
	}
}

func (p *peerConnection) onConnectionStateChange(state webrtc.PeerConnectionState) {
	log := log.Logger()
	log.Debug().
		Str("connection_state", state.String()).
		Msg("webRTC connection state changed")

	switch state {
	case webrtc.PeerConnectionStateNew:
	case webrtc.PeerConnectionStateConnecting:
	case webrtc.PeerConnectionStateConnected:
	case webrtc.PeerConnectionStateDisconnected:
		p.error(false, fmt.Errorf("webRTC connection disconnected"))
	case webrtc.PeerConnectionStateFailed:
		p.error(false, fmt.Errorf("webRTC connection failed"))
	case webrtc.PeerConnectionStateClosed:
		p.error(false, fmt.Errorf("webRTC connection closed"))
	}
}

func (p *peerConnection) onSignalingStateChange(state webrtc.SignalingState) {
	log := log.Logger()
	log.Debug().
		Str("signaling_state", state.String()).
		Msg("webRTC signaling state changed")

	switch state {
	case webrtc.SignalingStateStable:
	case webrtc.SignalingStateHaveLocalOffer:
	case webrtc.SignalingStateHaveRemoteOffer:
	case webrtc.SignalingStateHaveLocalPranswer:
	case webrtc.SignalingStateHaveRemotePranswer:
	case webrtc.SignalingStateClosed:
	}
}

func (p *peerConnection) onNegotiationNeeded() {
	log := log.Logger()
	log.Debug().
		Msg("webRTC peer connection negotiation needed")
	defer log.Debug().
		Msg("webRTC peer connection negotiation complete")

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
