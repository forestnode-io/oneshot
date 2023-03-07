package signallers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
)

type serverClientSignaller struct {
	url        string
	httpClient *http.Client
}

func NewServerClientSignaller(url string, client *http.Client) ClientSignaller {
	s := serverClientSignaller{
		url: url,
	}
	if client == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = client
	}

	return &s
}

func (s *serverClientSignaller) Start(ctx context.Context, handler OfferHandler) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request to signalling server failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http request to signalling server failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return fmt.Errorf("failed to close response body: %w", err)
	}

	var respStruct struct {
		SessionID int32
		Offer     string
	}
	if err := json.Unmarshal(body, &respStruct); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w\n%s", err, string(body))
	}

	answer, err := handler.HandleOffer(ctx, respStruct.SessionID, sdp.Offer(respStruct.Offer))
	if err != nil {
		return fmt.Errorf("failed to handle offer: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"Answer":    string(answer),
		"SessionID": respStruct.SessionID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal answer: %w", err)
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err = s.httpClient.Do(req)
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
