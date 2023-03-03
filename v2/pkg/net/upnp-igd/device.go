package upnpigd

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"net/url"
	"time"
)

type Device struct {
	uuid     string
	name     string
	services []Service
	url      *url.URL
	localIP  net.IP
}

func (d *Device) AddPortMapping(ctx context.Context, protocol string, externalPort, internalPort uint16, description string, leaseDuration uint32) error {
	req := AddPortMappingRequest{
		Protocol:     protocol,
		InternalPort: int(internalPort),
		ExternalPort: int(externalPort),
		Description:  description,
		Duration:     time.Duration(leaseDuration) * time.Second,
	}

	for _, s := range d.services {
		resp, err := s.Execute(ctx, &req)
		if err != nil {
			var env soapErrorResponse
			if e := xml.Unmarshal(resp, &env); e != nil {
				return fmt.Errorf("error unmarshaling error response: %w", e)
			}
			if env.ErrorCode == 725 {
				//TODO(raphaelreyna): handle error code 725 - OnlyPermanentLeasesSupported
				return fmt.Errorf("device only supports permanent leases")
			}
			return fmt.Errorf("error executing request: %w", err)
		}

	}

	return nil
}

func (d *Device) GetExternalIP(ctx context.Context) (net.IP, error) {
	var (
		respBytes []byte
		err       error
	)

	for _, s := range d.services {
		respBytes, err = s.Execute(ctx, &GetExternalIPRequest{})
		if err == nil {
			break
		}
	}

	var resp soapGetExternalIPAddressResponseEnvelope
	if err = xml.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	ip := net.ParseIP(resp.Body.GetExternalIPAddressResponse.NewExternalIPAddress)

	return ip, nil
}

func (d *Device) DeletePortMapping(ctx context.Context, protocol string, externalPort uint16) error {
	req := DeletePortMappingRequest{
		Protocol:     protocol,
		ExternalPort: int(externalPort),
	}

	for _, s := range d.services {
		_, err := s.Execute(ctx, &req)
		if err != nil {
			continue
		}
	}

	return nil
}
