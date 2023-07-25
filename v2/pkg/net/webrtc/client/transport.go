package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
	"github.com/forestnode-io/oneshot/v2/pkg/log"
	oneshotwebrtc "github.com/forestnode-io/oneshot/v2/pkg/net/webrtc"
	"github.com/forestnode-io/oneshot/v2/pkg/net/webrtc/sdp"
)

type dcBundle struct {
	dc  *webrtc.DataChannel
	raw datachannel.ReadWriteCloser
	err error
}

type Transport struct {
	config *webrtc.Configuration

	peerAddresses []string
	paMu          sync.Mutex

	peerConn              *webrtc.PeerConnection
	dcChan                chan dcBundle
	continueChan          chan struct{}
	connectionEstablished chan struct{}
}

func NewTransport(config *webrtc.Configuration) (*Transport, error) {
	log := log.Logger()
	t := Transport{
		config:                config,
		dcChan:                make(chan dcBundle, 1),
		continueChan:          make(chan struct{}, 1),
		connectionEstablished: make(chan struct{}, 1),
	}

	se := webrtc.SettingEngine{}
	se.DetachDataChannels()
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	pc, err := api.NewPeerConnection(*t.config)
	if err != nil {
		return nil, fmt.Errorf("unable to create new webRTC peer connection: %w", err)
	}
	t.peerConn = pc

	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		var opened bool
		d.OnOpen(func() {
			log.Debug().
				Msg("data channel opened")
			d.SetBufferedAmountLowThreshold(oneshotwebrtc.BufferedAmountLowThreshold)
			rawDC, err := d.Detach()
			if err != nil {
				t.dcChan <- dcBundle{err: err}
				close(t.dcChan)
				return
			}

			t.dcChan <- dcBundle{dc: d, raw: rawDC}
			close(t.dcChan)
			close(t.connectionEstablished)
		})
		d.OnBufferedAmountLow(func() {
			t.continueChan <- struct{}{}
		})
		d.OnClose(func() {
			log.Debug().
				Msg("data channel closed")
			if !opened {
				close(t.dcChan)
			}
		})
		d.OnError(func(err error) {
			t.dcChan <- dcBundle{err: err}
		})
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Debug().
			Str("state", state.String()).
			Msg("peer connection state changed")
		if state == webrtc.PeerConnectionStateDisconnected ||
			state == webrtc.PeerConnectionStateFailed {
			if err = t.peerConn.Close(); err != nil {
				log.Error().Err(err).
					Msg("unable to close peer connection")
			}
		}
	})

	t.peerAddresses = make([]string, 0)
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		t.paMu.Lock()
		defer t.paMu.Unlock()
		addr := fmt.Sprintf("%s:%d", c.Address, c.Port)
		t.peerAddresses = append(t.peerAddresses, addr)
		log.Debug().
			Str("addr", addr).
			Msg("ICE candidate")
	})

	return &t, nil
}

func (t *Transport) HandleOffer(ctx context.Context, id string, o sdp.Offer) (sdp.Answer, error) {
	pc := t.peerConn

	offer, err := o.WebRTCSessionDescription()
	if err != nil {
		return "", fmt.Errorf("unable to parse offer: %w", err)
	}

	if err := pc.SetRemoteDescription(offer); err != nil {
		return "", fmt.Errorf("unable to set remote description: %w", err)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return "", fmt.Errorf("unable to create answer: %w", err)
	}

	if err = pc.SetLocalDescription(answer); err != nil {
		return "", fmt.Errorf("unable to set local description: %w", err)
	}

	<-webrtc.GatheringCompletePromise(pc)

	return sdp.Answer(answer.SDP), nil
}

func (t *Transport) PeerAddresses() []string {
	t.paMu.Lock()
	defer t.paMu.Unlock()
	return t.peerAddresses
}

func (t *Transport) WaitForConnectionEstablished(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.connectionEstablished:
	}
	return nil
}
