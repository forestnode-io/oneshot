package client

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
	oneshotwebrtc "github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
)

type dcBundle struct {
	dc  *webrtc.DataChannel
	raw datachannel.ReadWriteCloser
	err error
}

type Transport struct {
	Config *webrtc.Configuration

	peerAddresses []string
	paMu          sync.Mutex

	peerConn              *webrtc.PeerConnection
	dcChan                chan dcBundle
	continueChan          chan struct{}
	connectionEstablished chan struct{}
}

func (t *Transport) HandleOffer(ctx context.Context, id int32, o sdp.Offer) (sdp.Answer, error) {
	se := webrtc.SettingEngine{}
	se.DetachDataChannels()
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	pc, err := api.NewPeerConnection(*t.Config)
	if err != nil {
		return "", fmt.Errorf("unable to create new webRTC peer connection: %w", err)
	}
	t.peerConn = pc

	t.dcChan = make(chan dcBundle, 1)
	t.continueChan = make(chan struct{}, 1)
	t.connectionEstablished = make(chan struct{})

	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnOpen(func() {
			d.SetBufferedAmountLowThreshold(oneshotwebrtc.BufferedAmountLowThreshold)
			rawDC, err := d.Detach()
			if err != nil {
				t.dcChan <- dcBundle{err: err}
				close(t.dcChan)
				return
			}
			t.dcChan <- dcBundle{dc: d, raw: rawDC}
			close(t.dcChan)
		})
		d.OnBufferedAmountLow(func() {
			t.continueChan <- struct{}{}
		})
		d.OnClose(func() {
			log.Println("data channel closed")
			close(t.dcChan)
		})
		d.OnError(func(err error) {
			t.dcChan <- dcBundle{err: err}
		})
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("peer connection state changed: %v", state)
		if state == webrtc.PeerConnectionStateDisconnected ||
			state == webrtc.PeerConnectionStateFailed {
			if err = t.peerConn.Close(); err != nil {
				log.Printf("unable to close peer connection: %v", err)
			}
		} else if state == webrtc.PeerConnectionStateConnected {
			close(t.connectionEstablished)
		}
	})

	t.peerAddresses = make([]string, 0)
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		t.paMu.Lock()
		defer t.paMu.Unlock()
		t.peerAddresses = append(t.peerAddresses, fmt.Sprintf("%s:%d", c.Address, c.Port))
	})

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

func (t *Transport) ConnectionEstablished() <-chan struct{} {
	return t.connectionEstablished
}
