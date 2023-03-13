package signallingserver

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/transport"
)

var id = "oneshot-signalling-server"

type oneshotServer struct {
	Arrival messages.ServerArrivalRequest
	done    chan struct{}
	msgConn *transport.Transport

	resetExpirationTimer func()

	slowDownHeartbeatTo    time.Duration
	slowDownHeartAfter     time.Duration
	slowDownHeartbeatTimer *time.Timer

	msgChan chan messages.Message
	errChan chan error

	mu sync.Mutex
}

func (o *oneshotServer) startReadPump(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			m, err := o.msgConn.Read()
			if err != nil {
				o.errChan <- err
				close(o.errChan)
				close(o.msgChan)
				return
			}

			_, ok := m.(*messages.Ping)
			if ok {
				o.resetExpirationTimer()
				continue
			}

			o.msgChan <- m
		}
	}()
}

func (o *oneshotServer) read() (messages.Message, error) {
	select {
	case msg := <-o.msgChan:
		return msg, nil
	case err := <-o.errChan:
		return nil, err
	}
}

func (o *oneshotServer) write(msg messages.Message) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.msgConn.Write(msg)
}

func newOneshotServer(ctx context.Context, requiredID string, conn net.Conn, requestURL func(string, bool) (string, error)) (*oneshotServer, error) {
	o := oneshotServer{
		msgConn:             transport.NewTransport(conn),
		done:                make(chan struct{}),
		slowDownHeartbeatTo: 8 * sdp.PingWindowDuration,
		slowDownHeartAfter:  6 * time.Second,
		msgChan:             make(chan messages.Message, 1),
		errChan:             make(chan error, 1),
	}

	o.startReadPump(ctx)

	// exchange version info
	m, err := o.read()
	if err != nil {
		return nil, fmt.Errorf("unable to read handshake: %w", err)
	}
	rh, ok := m.(*messages.Handshake)
	if !ok {
		return nil, messages.ErrInvalidRequestType
	}
	if rh.Error != nil {
		return nil, fmt.Errorf("error from remote: %s", rh.Error)
	}

	h := messages.Handshake{
		ID: id,
		VersionInfo: messages.VersionInfo{
			Version: "0.0.1",
		},
	}

	if rh.ID != requiredID && requiredID != "" {
		h.Error = fmt.Errorf("unautharized")
		err = o.write(&h)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("invalid id")
	}

	err = o.write(&h)
	if err != nil {
		return nil, fmt.Errorf("unable to write handshake: %w", err)
	}

	log.Printf("oneshot server version: %s", rh.VersionInfo.Version)

	// grab the arrival request and store it
	m, err = o.read()
	if err != nil {
		return nil, fmt.Errorf("unable to read arrival request: %w", err)
	}

	ar, ok := m.(*messages.ServerArrivalRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type, expected ArrivalRequest, got: %s", m.Type())
	}
	o.Arrival = *ar

	resp := messages.ServerArrivalResponse{}
	if rurl := ar.URL; rurl != nil {
		assignedURL, err := requestURL(rurl.URL, rurl.Required)
		if err != nil {
			return nil, fmt.Errorf("unable to assign requested url: %w", err)
		}
		resp.AssignedURL = assignedURL
	}

	if err = o.write(&resp); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	pwd := sdp.PingWindowDuration
	t := time.AfterFunc(pwd, func() {
		log.Printf("ping window expired")
		cancel()
	})
	o.resetExpirationTimer = func() {
		t.Reset(pwd)
	}

	o.slowDownHeartbeatTimer = time.AfterFunc(o.slowDownHeartAfter, func() {
		newPwd := o.slowDownHeartbeatTo
		diff := newPwd - pwd
		pwd = newPwd
		if diff > 0 {
			time.Sleep(diff)
		}
		o.write(&messages.UpdatePingRateRequest{
			Period: newPwd / 2,
		})
		log.Printf("slowed down heartbeat period to %+v", pwd)
	})

	return &o, nil
}

func (o *oneshotServer) RequestOffer(ctx context.Context, sessionID string) (sdp.Offer, error) {
	req := messages.GetOfferRequest{
		SessionID: sessionID,
	}
	if err := o.write(&req); err != nil {
		return "", err
	}

	resp, err := o.read()
	if err != nil {
		return "", err
	}

	gor, ok := resp.(*messages.GetOfferResponse)
	if !ok {
		return "", fmt.Errorf("invalid request type, expected GetOfferResponse, got: %s", resp.Type())
	}

	return sdp.Offer(gor.Offer), nil
}

func (o *oneshotServer) SendAnswer(ctx context.Context, sessionID string, answer sdp.Answer) error {
	defer func() {
		o.done <- struct{}{}
	}()

	req := messages.GotAnswerRequest{
		SessionID: sessionID,
		Answer:    string(answer),
	}
	if err := o.write(&req); err != nil {
		return fmt.Errorf("unable to send answer: %w", err)
	}

	resp, err := o.read()
	if err != nil {
		return err
	}

	gar, ok := resp.(*messages.GotAnswerResponse)
	if !ok {
		return fmt.Errorf("invalid request type, expected GetOfferResponse, got: %s", resp.Type())
	}

	return gar.Error
}

func (o *oneshotServer) Close() error {
	close(o.done)
	return o.msgConn.Close()
}

func (o *oneshotServer) Done() <-chan struct{} {
	return o.done
}
