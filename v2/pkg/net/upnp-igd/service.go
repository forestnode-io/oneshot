package upnpigd

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type Request interface {
	SOAP(urn string, localIP net.IP) (string, error)
	Function() string
}

type Service struct {
	ID      string
	URL     string
	URN     string
	LocalIP net.IP

	Client    *http.Client `json:"-"`
	UserAgent string
}

func (s *Service) Execute(ctx context.Context, r Request) ([]byte, error) {
	if s.URN == "" {
		return nil, fmt.Errorf("invalid urn: %s", s.URN)
	}

	body, err := r.SOAP(s.URN, s.LocalIP)
	if err != nil {
		return nil, fmt.Errorf("error creating request body: %w", err)
	}

	req := request{
		url:  s.URL,
		body: body,
		header: http.Header{
			"User-Agent": []string{s.UserAgent},
		},
		service:  s.URN,
		function: r.Function(),
	}
	respBytes, err := req.do(ctx, s.Client)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	return respBytes, nil
}

type GetExternalIPRequest struct{}

func (r *GetExternalIPRequest) SOAP(urn string, _ net.IP) (string, error) {
	tmplt := `<u:GetExternalIPAddress xmlns:u="%s" />`
	return fmt.Sprintf(tmplt, urn), nil
}

func (r *GetExternalIPRequest) Function() string {
	return "GetExternalIPAddress"
}

type AddPortMappingRequest struct {
	Protocol     string
	InternalPort int
	ExternalPort int
	Description  string
	Duration     time.Duration
}

func (r *AddPortMappingRequest) Function() string {
	return "AddPortMapping"
}

func (r *AddPortMappingRequest) SOAP(urn string, localIP net.IP) (string, error) {
	tmplt := `<u:AddPortMapping xmlns:u="%s">
	<NewRemoteHost></NewRemoteHost>
	<NewExternalPort>%d</NewExternalPort>
	<NewProtocol>%s</NewProtocol>
	<NewInternalPort>%d</NewInternalPort>
	<NewInternalClient>%s</NewInternalClient>
	<NewEnabled>1</NewEnabled>
	<NewPortMappingDescription>%s</NewPortMappingDescription>
	<NewLeaseDuration>%d</NewLeaseDuration>
	</u:AddPortMapping>`

	r.Protocol = strings.ToUpper(r.Protocol)
	if r.Protocol != "TCP" && r.Protocol != "UDP" || r.Protocol == "" {
		return "", fmt.Errorf("invalid protocol: %s", r.Protocol)
	}

	if r.InternalPort == 0 {
		return "", fmt.Errorf("invalid internal port: %d", r.InternalPort)
	}

	if r.ExternalPort == 0 {
		return "", fmt.Errorf("invalid external port: %d", r.ExternalPort)
	}

	if r.Description == "" {
		return "", fmt.Errorf("invalid description: %s", r.Description)
	}

	if r.Duration == 0 {
		return "", fmt.Errorf("invalid duration: %s", r.Duration)
	}

	if urn == "" {
		return "", fmt.Errorf("invalid urn: %s", urn)
	}

	if localIP == nil {
		return "", fmt.Errorf("invalid local ip: %s", localIP)
	}

	return fmt.Sprintf(tmplt,
		urn,
		r.ExternalPort,
		r.Protocol,
		r.InternalPort,
		localIP,
		r.Description,
		r.Duration/time.Second,
	), nil
}

type DeletePortMappingRequest struct {
	Protocol     string
	ExternalPort int
}

func (r *DeletePortMappingRequest) Function() string {
	return "DeletePortMapping"
}

func (r *DeletePortMappingRequest) SOAP(urn string, _ net.IP) (string, error) {
	tmplt := `<u:DeletePortMapping xmlns:u="%s">
	<NewRemoteHost></NewRemoteHost>
	<NewExternalPort>%d</NewExternalPort>
	<NewProtocol>%s</NewProtocol>
	</u:DeletePortMapping>`

	r.Protocol = strings.ToUpper(r.Protocol)
	if r.Protocol != "TCP" && r.Protocol != "UDP" || r.Protocol == "" {
		return "", fmt.Errorf("invalid protocol: %s", r.Protocol)
	}

	if r.ExternalPort == 0 {
		return "", fmt.Errorf("invalid external port: %d", r.ExternalPort)
	}

	if urn == "" {
		return "", fmt.Errorf("invalid urn: %s", urn)
	}

	return fmt.Sprintf(tmplt,
		urn,
		r.ExternalPort,
		r.Protocol,
	), nil
}

type soapGetExternalIPAddressResponseEnvelope struct {
	XMLName xml.Name
	Body    soapGetExternalIPAddressResponseBody `xml:"Body"`
}

type soapGetExternalIPAddressResponseBody struct {
	XMLName                      xml.Name
	GetExternalIPAddressResponse getExternalIPAddressResponse `xml:"GetExternalIPAddressResponse"`
}

type getExternalIPAddressResponse struct {
	NewExternalIPAddress string `xml:"NewExternalIPAddress"`
}

type soapErrorResponse struct {
	ErrorCode        int    `xml:"Body>Fault>detail>UPnPError>errorCode"`
	ErrorDescription string `xml:"Body>Fault>detail>UPnPError>errorDescription"`
}
