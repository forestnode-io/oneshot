package signallers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
)

type serverClientSignaller struct {
	url        string
	httpClient *http.Client
	offer      sdp.Offer
	sessionID  string
}

func NewServerClientSignaller(url, sessionID string, offer sdp.Offer, client *http.Client) ClientSignaller {
	s := serverClientSignaller{
		url:       url,
		sessionID: sessionID,
		offer:     offer,
	}
	if client == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = client
	}

	return &s
}

func (s *serverClientSignaller) Start(ctx context.Context, handler OfferHandler) error {
	answer, err := handler.HandleOffer(ctx, s.sessionID, s.offer)
	if err != nil {
		return fmt.Errorf("failed to handle offer: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"Answer":    string(answer),
		"SessionID": s.sessionID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal answer: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request to signalling server failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http request to signalling server failed: %s", resp.Status)
	}

	return nil
}

func (s *serverClientSignaller) Shutdown() error {
	return nil
}
