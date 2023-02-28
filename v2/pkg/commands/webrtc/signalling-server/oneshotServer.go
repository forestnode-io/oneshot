package signallingserver

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/transport"
)

type oneshotServer struct {
	Arrival messages.ArrivalRequest
	done    chan struct{}
	msgConn *transport.Transport
}

func newOneshotServer(conn net.Conn) (*oneshotServer, error) {
	o := oneshotServer{
		msgConn: transport.NewTransport(conn),
		done:    make(chan struct{}),
	}

	// exchange version info
	thisVi := messages.VersionInfo{
		Version: "0.0.1",
	}
	err := o.msgConn.Write(&thisVi)
	if err != nil {
		return nil, err
	}

	m, err := o.msgConn.Read()
	if err != nil {
		return nil, err
	}

	vi, ok := m.(*messages.VersionInfo)
	if !ok {
		return nil, messages.ErrInvalidRequestType
	}

	log.Printf("oneshot server version: %s", vi.Version)

	// grab the arrival request and store it
	m, err = o.msgConn.Read()
	if err != nil {
		return nil, fmt.Errorf("unable to read arrival request: %w", err)
	}

	ar, ok := m.(*messages.ArrivalRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type, expected ArrivalRequest, got: %s", m.Type())
	}
	o.Arrival = *ar

	resp := messages.ArrivalResponse{}
	if err = o.msgConn.Write(&resp); err != nil {
		return nil, err
	}

	return &o, nil
}

func (o *oneshotServer) RequestOffer(ctx context.Context, sessionID int32) (sdp.Offer, error) {
	req := messages.GetOfferRequest{
		SessionID: sessionID,
	}
	if err := o.msgConn.Write(&req); err != nil {
		return "", err
	}

	resp, err := o.msgConn.Read()
	if err != nil {
		return "", err
	}

	gor, ok := resp.(*messages.GetOfferResponse)
	if !ok {
		return "", messages.ErrInvalidRequestType
	}

	return sdp.Offer(gor.Offer), nil
}

func (o *oneshotServer) SendAnswer(ctx context.Context, sessionID int32, answer sdp.Answer) error {
	req := messages.GotAnswerRequest{
		SessionID: sessionID,
		Answer:    string(answer),
	}
	if err := o.msgConn.Write(&req); err != nil {
		return fmt.Errorf("unable to send answer: %w", err)
	}

	resp, err := o.msgConn.Read()
	if err != nil {
		return fmt.Errorf("unable to read answer response: %w", err)
	}

	gar, ok := resp.(*messages.GotAnswerResponse)
	if !ok {
		return fmt.Errorf("invalid request type, expected GotAnswerResponse, got: %s", resp.Type())
	}

	if gar.SessionID != sessionID {
		return fmt.Errorf("session id mismatch: %d != %d", gar.SessionID, sessionID)
	}

	o.done <- struct{}{}

	return gar.Error
}

func (o *oneshotServer) Close() error {
	close(o.done)
	return o.msgConn.Close()
}

func (o *oneshotServer) Done() <-chan struct{} {
	return o.done
}
