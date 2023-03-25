package messages

import (
	"encoding/json"
	"fmt"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/proto"
)

var ErrInvalidRequestType = fmt.Errorf("invalid request type")

func Unmarshal(typeName string, data []byte) (Message, error) {
	switch typeName {
	case "Handshake":
		var h Handshake
		err := json.Unmarshal(data, &h)
		return &h, err
	case "ServerArrivalRequest":
		var a ServerArrivalRequest
		err := json.Unmarshal(data, &a)
		return &a, err
	case "ServerArrivalResponse":
		var a ServerArrivalResponse
		err := json.Unmarshal(data, &a)
		return &a, err
	case "GetOfferRequest":
		var g GetOfferRequest
		err := json.Unmarshal(data, &g)
		return &g, err
	case "GetOfferResponse":
		var g GetOfferResponse
		err := json.Unmarshal(data, &g)
		return &g, err
	case "GotAnswerRequest":
		var g GotAnswerRequest
		err := json.Unmarshal(data, &g)
		return &g, err
	case "GotAnswerResponse":
		var g GotAnswerResponse
		err := json.Unmarshal(data, &g)
		return &g, err
	case "ClientArrivalRequest":
		var a ClientArrivalRequest
		err := json.Unmarshal(data, &a)
		return &a, err
	case "ClientArrivalResponse":
		var a ClientArrivalResponse
		err := json.Unmarshal(data, &a)
		return &a, err
	case "AnswerOfferRequest":
		var a AnswerOfferRequest
		err := json.Unmarshal(data, &a)
		return &a, err
	case "AnswerOfferResponse":
		var a AnswerOfferResponse
		err := json.Unmarshal(data, &a)
		return &a, err
	case "Ping":
		var p Ping
		return &p, nil
	case "UpdatePingRateRequest":
		var u UpdatePingRateRequest
		err := json.Unmarshal(data, &u)
		return &u, err
	}

	return nil, fmt.Errorf("unknown message type: %s", typeName)
}

func FromRPCEnvelope(env *proto.Envelope) (Message, error) {
	return Unmarshal(env.Type, env.Data)
}

func ToRPCEnvelope(msg Message) (*proto.Envelope, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return &proto.Envelope{
		Type: msg.Type(),
		Data: data,
	}, nil
}
