package messages

import (
	"encoding/json"
	"fmt"
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
	}

	return nil, fmt.Errorf("unknown message type: %s", typeName)
}
