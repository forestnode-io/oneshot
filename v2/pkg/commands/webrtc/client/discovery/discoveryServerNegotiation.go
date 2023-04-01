package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	signallingserver "github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/signalling-server"
)

func NegotiateOfferRequest(ctx context.Context, url string, client *http.Client) (*signallingserver.ClientOfferRequestResponse, error) {
	// perform the first request which saves our spot in the queue.
	// we're going to use the same pathways as browser clients to we
	// set the accept header to text/http and the user agent to oneshot.
	// the server will respond differently based on the user agent, it wont send html.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "oneshot")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get token request response: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token request response: %w", err)
	}

	bodyParts := strings.Split(string(body), "\n")
	if len(bodyParts) != 2 {
		return nil, fmt.Errorf("invalid token request response: %s", string(body))
	}
	reqURL := bodyParts[0]
	token := bodyParts[1]

	req, err = http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create offer request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Session-Token", token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request offer response: %w", err)
	}
	defer resp.Body.Close()

	var corr signallingserver.ClientOfferRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&corr); err != nil {
		return nil, fmt.Errorf("failed to decode offer request response: %w", err)
	}
	return &corr, nil
}
