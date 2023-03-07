package sdp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pion/webrtc/v3"
)

type Offer string

func OfferFromJSON(data []byte) (Offer, error) {
	sdp := webrtc.SessionDescription{}
	if err := json.Unmarshal(data, &sdp); err != nil {
		return "", err
	}
	if sdp.Type != webrtc.SDPTypeOffer {
		return "", fmt.Errorf("invalid SDP type: %s", sdp.Type)
	}
	return Offer(sdp.SDP), nil
}

func (o Offer) MarshalJSON() ([]byte, error) {
	sdp, err := o.WebRTCSessionDescription()
	if err != nil {
		return nil, err
	}
	return json.Marshal(sdp)
}

func (s Offer) WebRTCSessionDescription() (webrtc.SessionDescription, error) {
	sdp := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  string(s),
	}
	_, err := sdp.Unmarshal()
	return sdp, err
}

type Answer string

func AnswerFromJSON(data []byte) (Answer, error) {
	sdp := webrtc.SessionDescription{}
	if err := json.Unmarshal(data, &sdp); err != nil {
		return "", err
	}
	if sdp.Type != webrtc.SDPTypeAnswer {
		return "", fmt.Errorf("invalid SDP type: %s", sdp.Type)
	}
	return Answer(sdp.SDP), nil
}

func (s Answer) WebRTCSessionDescription() (webrtc.SessionDescription, error) {
	sdp := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  string(s),
	}
	_, err := sdp.Unmarshal()
	return sdp, err
}

func (s Answer) MarshalJSON() ([]byte, error) {
	sdp, err := s.WebRTCSessionDescription()
	if err != nil {
		return nil, err
	}
	return json.Marshal(sdp)
}

const PingWindowDuration = 500 * time.Millisecond
