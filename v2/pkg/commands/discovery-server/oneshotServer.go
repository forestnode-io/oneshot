package discoveryserver

import (
	"context"
	"fmt"

	pionwebrtc "github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/proto"
	"github.com/raphaelreyna/oneshot/v2/pkg/version"
	"github.com/rs/zerolog"
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
	var (
		log = zerolog.Ctx(ctx)
		o   = oneshotServer{
			done:         make(chan struct{}),
			stream:       stream,
			msgChan:      make(chan messages.Message, 1),
			errChan:      make(chan error, 1),
			resetPending: resetPending,
		}
	)

	// exchange version info
	handshake, err := receive[*messages.Handshake](stream)
	if err != nil {
		return nil, fmt.Errorf("unable to read handshake: %w", err)
	}
	if handshake.Error != "" {
		return nil, fmt.Errorf("error from remote: %s", handshake.Error)
	}

	log.Info().
		Str("version", handshake.VersionInfo.Version).
		Str("api-version", handshake.VersionInfo.APIVersion).
		Str("id", handshake.ID).
		Msg("received handshake")

	responseHandshake := messages.Handshake{
		ID: id,
		VersionInfo: messages.VersionInfo{
			Version:    version.Version,
			APIVersion: version.APIVersion,
		},
	}

	if responseHandshake.ID != requiredID && requiredID != "" {
		responseHandshake.Error = "unauthorized"
		if err := send(stream, &responseHandshake); err != nil {
			log.Error().Err(err).
				Msg("unable to write handshake")
		}

		return nil, fmt.Errorf("invalid id")
	}

	if err = send(stream, &responseHandshake); err != nil {
		return nil, fmt.Errorf("unable to write handshake: %w", err)
	}

	log.Debug().
		Str("version", responseHandshake.VersionInfo.Version).
		Str("api-version", responseHandshake.VersionInfo.APIVersion).
		Str("id", responseHandshake.ID).
		Msg("sent handshake")

	// grab the arrival request and store it
	arrivalRequest, err := receive[*messages.ServerArrivalRequest](stream)
	if err != nil {
		return nil, fmt.Errorf("unable to read arrival request: %w", err)
	}
	o.Arrival = *arrivalRequest

	log.Debug().
		Str("url", arrivalRequest.URL.URL).
		Bool("url-required", arrivalRequest.URL.Required).
		Msg("received arrival request")

	resp := messages.ServerArrivalResponse{}
	rurl := ""
	rurlRequired := false
	if arrivalRequest.URL != nil {
		rurl = arrivalRequest.URL.URL
		rurlRequired = arrivalRequest.URL.Required
	}
	assignedURL, err := requestURL(rurl, rurlRequired)
	if err != nil {
		return nil, fmt.Errorf("unable to assign requested url: %w", err)
	}
	resp.AssignedURL = assignedURL

	if err = send(stream, &resp); err != nil {
		return nil, fmt.Errorf("unable to write arrival response: %w", err)
	}

	log.Info().
		Str("assigned-url", assignedURL).
		Msg("assigned URL")

	return &o, nil
}

func (o *oneshotServer) RequestOffer(ctx context.Context, sessionID string, conf *pionwebrtc.Configuration) (sdp.Offer, error) {
	req := messages.GetOfferRequest{
		SessionID:     sessionID,
		Configuration: conf,
	}

	if err := send(o.stream, &req); err != nil {
		return "", fmt.Errorf("unable to write offer request: %w", err)
	}

	gor, err := receive[*messages.GetOfferResponse](o.stream)
	if err != nil {
		return "", fmt.Errorf("unable to read offer response: %w", err)
	}

	return sdp.Offer(gor.Offer), nil
}

func (o *oneshotServer) SendAnswer(ctx context.Context, sessionID string, answer sdp.Answer) error {
	req := messages.GotAnswerRequest{
		SessionID: sessionID,
		Answer:    string(answer),
	}

	if err := send(o.stream, &req); err != nil {
		return fmt.Errorf("unable to write answer: %w", err)
	}

	gar, err := receive[*messages.GotAnswerResponse](o.stream)
	if err != nil {
		return fmt.Errorf("unable to read answer response: %w", err)
	}

	go func() {
		log := zerolog.Ctx(ctx)
		fsr, err := receive[*messages.FinishedSessionRequest](o.stream)
		if err != nil {
			statErr, ok := status.FromError(err)
			if !ok || (ok && statErr.Code() != codes.Canceled) {
				log.Error().Err(err).
					Msg("error receiving message")
			}
			return
		}

		if fsr.Error != "" {
			log.Warn().Err(err).
				Msg("session failed")
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

func send(stream proto.SignallingServer_ConnectServer, m messages.Message) error {
	env, err := messages.ToRPCEnvelope(m)
	if err != nil {
		return fmt.Errorf("unable to marshal message: %w", err)
	}
	return stream.Send(env)
}

func receive[M messages.Message](stream proto.SignallingServer_ConnectServer) (M, error) {
	var (
		m  M
		ok bool
	)

	env, err := stream.Recv()
	if err != nil {
		return m, fmt.Errorf("unable to read message: %w", err)
	}

	msg, err := messages.FromRPCEnvelope(env)
	if err != nil {
		return m, fmt.Errorf("unable to read message: %w", err)
	}

	m, ok = msg.(M)
	if !ok {
		return m, fmt.Errorf("invalid message type, expected %T, got %T", m, msg)
	}

	return m, nil
}
