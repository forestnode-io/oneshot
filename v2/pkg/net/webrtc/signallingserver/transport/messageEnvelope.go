package transport

import (
	"encoding/json"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
)

type envelope struct {
	Type string
	Data json.RawMessage
}

func newEnvelope(m messages.Message) (*envelope, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return &envelope{
		Type: m.Type(),
		Data: data,
	}, nil
}

func (m *envelope) message() (messages.Message, error) {
	return messages.Unmarshal(m.Type, m.Data)
}
