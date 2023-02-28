package messages

import (
	"encoding/json"
	"fmt"
)

var ErrInvalidRequestType = fmt.Errorf("invalid request type")

func Unmarshal(typeName string, data []byte) (Message, error) {
	switch typeName {
	case "VersionInfo":
		var v VersionInfo
		err := json.Unmarshal(data, &v)
		return &v, err
	case "ArrivalRequest":
		var a ArrivalRequest
		err := json.Unmarshal(data, &a)
		return &a, err
	case "ArrivalResponse":
		var a ArrivalResponse
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
	}

	return nil, fmt.Errorf("unknown message type: %s", typeName)
}
