package webrtc

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/pion/webrtc/v3"
)

type SDExchange func(context.Context, int, *webrtc.SessionDescription) (*webrtc.SessionDescription, error)

type SessionSignaller interface {
	// ResisterServerSD registers this oneshot server with the signaling server.
	// The returned channel receives when a client wants to connect to this server.
	RegisterServer(context.Context, string) (<-chan int, error)
	ExchangeSD(context.Context, int, *webrtc.SessionDescription) (*webrtc.SessionDescription, error)
}

type WebRTCAgent struct {
	SignallingServer SessionSignaller
	ICEServerURL     string
	Handler          http.HandlerFunc
}

func (a *WebRTCAgent) Run(ctx context.Context, id string) error {
	clientsChan, err := a.SignallingServer.RegisterServer(ctx, id)
	if err != nil {
		return err
	}

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{a.ICEServerURL},
			},
		},
	}

	for clientRequestID := range clientsChan {
		err := func(clientRequestID int) error {
			errs := make(chan error, 1)
			c := configuration{
				Configuration:   &config,
				ClientRequestID: clientRequestID,
				SDExchange:      a.SignallingServer.ExchangeSD,
			}

			pc, err := newPeerConnection(ctx, errs, c)
			if err != nil {
				return err
			}
			defer pc.Close()

			dc, err := newDataChannel("", errs, pc, a.Handler)
			if err != nil {
				log.Fatalf("unable to create data channel for webRTC peer connection: %s", err.Error())
			}
			defer dc.Close()

			for err := range errs {
				fmt.Printf("POOP: %s\n", err.Error())
			}

			return nil
		}(clientRequestID)
		if err != nil {
			continue
		}
	}

	return nil
}
