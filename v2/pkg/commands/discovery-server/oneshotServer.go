package discoveryserver

import (
	"context"
	"fmt"
	"log"

	pionwebrtc "github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/proto"
	"github.com/raphaelreyna/oneshot/v2/pkg/version"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var id = "oneshot-signalling-server"

type oneshotServer struct {
	Arrival messages.ServerArrivalRequest
	done    chan struct{}

	msgChan chan messages.Message
	errChan chan error

	resetPending func()

	stream proto.SignallingServer_ConnectServer
}

func newOneshotServer(ctx context.Context, requiredID string, stream proto.SignallingServer_ConnectServer, resetPending func(), requestURL func(string, bool) (string, error)) (*oneshotServer, error) {
	o := oneshotServer{
		done:         make(chan struct{}),
		stream:       stream,
		msgChan:      make(chan messages.Message, 1),
		errChan:      make(chan error, 1),
		resetPending: resetPending,
	}

	// exchange version info
	env, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("unable to read handshake: %w", err)
	}
	m, err := messages.FromRPCEnvelope(env)
	if err != nil {
		return nil, fmt.Errorf("unable to read handshake: %w", err)
	}

	rh, ok := m.(*messages.Handshake)
	if !ok {
		return nil, messages.ErrInvalidRequestType
	}
	if rh.Error != "" {
		return nil, fmt.Errorf("error from remote: %s", rh.Error)
	}
	log.Printf("oneshot client version: %s", rh.VersionInfo.Version)

	h := messages.Handshake{
		ID: id,
		VersionInfo: messages.VersionInfo{
			Version:    version.Version,
			APIVersion: version.APIVersion,
		},
	}

	if rh.ID != requiredID && requiredID != "" {
		h.Error = "unautharized"
		env, err = messages.ToRPCEnvelope(&h)
		if err != nil {
			return nil, err
		}
		if err = stream.Send(env); err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("invalid id")
	}

	env, err = messages.ToRPCEnvelope(&h)
	if err != nil {
		return nil, err
	}
	if err = stream.Send(env); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("unable to write handshake: %w", err)
	}

	log.Printf("oneshot server version: %s", rh.VersionInfo.Version)

	// grab the arrival request and store it
	env, err = stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("unable to read arrival request: %w", err)
	}
	m, err = messages.FromRPCEnvelope(env)
	if err != nil {
		return nil, fmt.Errorf("unable to read arrival request: %w", err)
	}

	ar, ok := m.(*messages.ServerArrivalRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type, expected ArrivalRequest, got: %s", m.Type())
	}
	o.Arrival = *ar

	resp := messages.ServerArrivalResponse{}
	rurl := ""
	rurlRequired := false
	if ar.URL != nil {
		rurl = ar.URL.URL
		rurlRequired = ar.URL.Required
	}
	assignedURL, err := requestURL(rurl, rurlRequired)
	if err != nil {
		return nil, fmt.Errorf("unable to assign requested url: %w", err)
	}
	resp.AssignedURL = assignedURL
	log.Printf("assigned url: %s", assignedURL)

	env, err = messages.ToRPCEnvelope(&resp)
	if err != nil {
		return nil, err
	}
	if err = stream.Send(env); err != nil {
		return nil, err
	}

	return &o, nil
}

func (o *oneshotServer) RequestOffer(ctx context.Context, sessionID string, conf *pionwebrtc.Configuration) (sdp.Offer, error) {
	req := messages.GetOfferRequest{
		SessionID:     sessionID,
		Configuration: conf,
	}

	env, err := messages.ToRPCEnvelope(&req)
	if err != nil {
		return "", err
	}
	if err = o.stream.Send(env); err != nil {
		return "", err
	}

	env, err = o.stream.Recv()
	if err != nil {
		return "", fmt.Errorf("unable to read offer: %w", err)
	}
	m, err := messages.FromRPCEnvelope(env)
	if err != nil {
		return "", fmt.Errorf("unable to read offer: %w", err)
	}

	gor, ok := m.(*messages.GetOfferResponse)
	if !ok {
		return "", fmt.Errorf("invalid request type, expected GetOfferResponse, got: %s", m.Type())
	}

	return sdp.Offer(gor.Offer), nil
}

func (o *oneshotServer) SendAnswer(ctx context.Context, sessionID string, answer sdp.Answer) error {
	req := messages.GotAnswerRequest{
		SessionID: sessionID,
		Answer:    string(answer),
	}
	env, err := messages.ToRPCEnvelope(&req)
	if err != nil {
		return err
	}
	if err = o.stream.Send(env); err != nil {
		return err
	}

	env, err = o.stream.Recv()
	if err != nil {
		return fmt.Errorf("unable to read offer: %w", err)
	}
	m, err := messages.FromRPCEnvelope(env)
	if err != nil {
		return fmt.Errorf("unable to read offer: %w", err)
	}

	gar, ok := m.(*messages.GotAnswerResponse)
	if !ok {
		return fmt.Errorf("invalid request type, expected GetOfferResponse, got: %s", m.Type())
	}

	go func() {
		env, err := o.stream.Recv()
		if err != nil {
			statErr, ok := status.FromError(err)
			if !ok || (ok && statErr.Code() != codes.Canceled) {
				log.Printf("error receiving message: %v", err)
			}
			return
		}

		m, err := messages.FromRPCEnvelope(env)
		if err != nil {
			log.Printf("error parsing message: %v", err)
			return
		}

		fsr, ok := m.(*messages.FinishedSessionRequest)
		if !ok {
			log.Printf("unexpected message type: %T", m)
			return
		}

		if fsr.Error != "" {
			log.Printf("session failed: %s", fsr.Error)
		}
		o.resetPending()
	}()

	if gar.Error == "" {
		return nil
	}

	return fmt.Errorf("session failed: %s", gar.Error)
}

func (o *oneshotServer) Close() {
	close(o.done)
}

func (o *oneshotServer) Done() <-chan struct{} {
	return o.done
}
