package sdp

import (
	"context"
	"fmt"
	"os"
)

type fileClientSignaller struct {
	offerFilePath  string
	answerFilePath string
}

func NewFileClientSignaller(offerFilePath, answerFilePath string) ClientSignaller {
	return &fileClientSignaller{
		offerFilePath:  offerFilePath,
		answerFilePath: answerFilePath,
	}
}

func (s *fileClientSignaller) Start(ctx context.Context, offerHandler OfferHandler) error {
	offerFileBytes, err := os.ReadFile(s.offerFilePath)
	if err != nil {
		return fmt.Errorf("failed to read offer file: %w", err)
	}

	offer, err := OfferFromJSON(offerFileBytes)
	if err != nil {
		return fmt.Errorf("failed to parse offer: %w", err)
	}

	answer, err := offerHandler.HandleOffer(ctx, offer)
	if err != nil {
		return err
	}

	answerJSON, err := answer.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal answer: %w", err)
	}

	answerFilePath := s.answerFilePath
	if answerFilePath == "" {
		answerFilePath = fmt.Sprintf("%s.answer", s.offerFilePath)
	}

	if err := os.WriteFile(answerFilePath, answerJSON, 0644); err != nil {
		return fmt.Errorf("failed to write answer file: %w", err)
	}

	return nil
}

func (s *fileClientSignaller) Shutdown() error {
	return nil
}
