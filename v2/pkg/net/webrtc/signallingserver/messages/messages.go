package messages

type Message interface {
	Type() string
}

type VersionInfo struct {
	APIVersion string
	Version    string
}

func (v *VersionInfo) Type() string {
	return "VersionInfo"
}

type BasicAuth struct {
	UsernameHash []byte
	PasswordHash []byte
}

type SessionURLRequest struct {
	URL      string
	Required bool
}

// sent from the oneshot server to the signalling server after VersionInfo has been exchanged
type ArrivalRequest struct {
	ID        string
	BasicAuth *BasicAuth
	URL       *SessionURLRequest
}

func (a *ArrivalRequest) Type() string {
	return "ArrivalRequest"
}

// sent from the signalling server to the oneshot server when it first connects in response to an ArrivalRequest
type ArrivalResponse struct {
	AssignedURL string
	Error       string
}

func (a *ArrivalResponse) Type() string {
	return "ArrivalResponse"
}

// sent from the signalling server to the oneshot server when a new session has been request by a client
type GetOfferRequest struct {
	SessionID int32
}

func (g *GetOfferRequest) Type() string {
	return "GetOfferRequest"
}

// sent from the oneshot server to the signalling server when it has crafted an offer for the client requesting a session
type GetOfferResponse struct {
	SessionID int32
	Offer     string
}

func (g *GetOfferResponse) Type() string {
	return "GetOfferResponse"
}

// sent from the signalling server to the oneshot server when a client has answered the offer
type GotAnswerRequest struct {
	SessionID int32
	Answer    string
}

func (g *GotAnswerRequest) Type() string {
	return "GotAnswerRequest"
}

// sent from the oneshot server to the signalling server when it has accepted the answer and started the session
type GotAnswerResponse struct {
	SessionID int32
	Error     error
}

func (g *GotAnswerResponse) Type() string {
	return "GotAnswerResponse"
}

// sent from the oneshot server to the signalling server when a session has ended
type FinishedSessionRequest struct {
	SessionID int32
	Error     error
}

func (f *FinishedSessionRequest) Type() string {
	return "FinishedSessionRequest"
}

// sent from the signalling server to the oneshot server when it has received the FinishedSessionRequest
type FinishedSessionResponse struct {
	SessionID int32
	Error     error
}

func (f *FinishedSessionResponse) Type() string {
	return "FinishedSessionResponse"
}
