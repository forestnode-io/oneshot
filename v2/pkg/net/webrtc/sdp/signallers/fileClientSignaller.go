package signallers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pion/webrtc/v3"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/sdp"
)

type fileClientSignaller struct {
	offer          sdp.Offer
	answerFilePath string
}

func NewFileClientSignaller(offerFilePath, answerFilePath string) (ClientSignaller, string, error) {
	offerFileBytes, err := os.ReadFile(offerFilePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read offer file: %w", err)
	}

	wsdp := webrtc.SessionDescription{}
	if err := json.Unmarshal(offerFileBytes, &wsdp); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal offer: %w", err)
	}
	wssdp, err := wsdp.Unmarshal()
	if err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal offer: %w", err)
	}
	var bat string
	for _, attribute := range wssdp.Attributes {
		if attribute.Key == "BasicAuthToken" {
			bat = attribute.Value
			break
		}
	}

	return &fileClientSignaller{
		offer:          sdp.Offer(wsdp.SDP),
		answerFilePath: answerFilePath,
	}, bat, nil
}

func (s *fileClientSignaller) Start(ctx context.Context, offerHandler OfferHandler) error {
	answer, err := offerHandler.HandleOffer(ctx, "", s.offer)
	if err != nil {
		return err
	}

	answerJSON, err := answer.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal answer: %w", err)
	}

	if err := os.WriteFile(s.answerFilePath, answerJSON, 0644); err != nil {
		return fmt.Errorf("failed to write answer file: %w", err)
	}

	return nil
}

func (s *fileClientSignaller) Shutdown() error {
	return nil
}
