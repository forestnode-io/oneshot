package upnpigd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Device struct {
	UUID     string
	Name     string
	Services []Service
	URL      *url.URL
}

func (d *Device) AddPortMapping(ctx context.Context, protocol string, externalPort, internalPort int, description string, leaseDuration time.Duration) error {
	req := AddPortMappingRequest{
		Protocol:     protocol,
		InternalPort: int(internalPort),
		ExternalPort: int(externalPort),
		Description:  description,
		Duration:     leaseDuration,
	}

	for _, s := range d.Services {
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

	for _, s := range d.Services {
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

func (d *Device) DeletePortMapping(ctx context.Context, protocol string, externalPort int) error {
	req := DeletePortMappingRequest{
		Protocol:     protocol,
		ExternalPort: externalPort,
	}

	for _, s := range d.Services {
		_, err := s.Execute(ctx, &req)
		if err != nil {
			continue
		}
	}

	return nil
}

func NewDevice(client *http.Client, st string, raw []byte) (*Device, error) {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(raw)), nil)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	respST := resp.Header.Get("St")
	if respST != st {
		return nil, fmt.Errorf("invalid st: %s", respST)
	}

	respLocation := resp.Header.Get("Location")
	if respLocation == "" {
		return nil, fmt.Errorf("invalid location: %s", respLocation)
	}

	respLocationURL, err := url.Parse(respLocation)
	if err != nil {
		return nil, fmt.Errorf("error parsing location url: %w", err)
	}

	respUSN := resp.Header.Get("USN")
	if respUSN == "" {
		return nil, fmt.Errorf("invalid usn: %s", respUSN)
	}

	respUUID := strings.TrimPrefix(strings.Split(respUSN, "::")[0], "uuid:")

	// get the description from the location
	resp, err = client.Get(respLocation)
	if err != nil {
		return nil, fmt.Errorf("error getting description: %w", err)
	}
	defer resp.Body.Close()

	if 400 <= resp.StatusCode {
		return nil, fmt.Errorf("error getting description: %s", resp.Status)
	}

	var r root
	err = xml.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return nil, fmt.Errorf("error decoding description: %w", err)
	}

	localIPAddr, err := localIP(respLocationURL)
	if err != nil {
		return nil, fmt.Errorf("error getting local ip: %w", err)
	}

	services, err := getServiceDescriptions(respLocation, r.Device, localIPAddr, client)
	if err != nil {
		return nil, fmt.Errorf("error getting service descriptions: %w", err)
	}

	return &Device{
		UUID:     respUUID,
		Name:     r.Device.FriendlyName,
		Services: services,
		URL:      respLocationURL,
	}, nil
}

func replaceRawPath(u *url.URL, rp string) {
	asURL, err := url.Parse(rp)
	if err != nil {
		return
	} else if asURL.IsAbs() {
		u.Path = asURL.Path
		u.RawQuery = asURL.RawQuery
	} else {
		var p, q string
		fs := strings.Split(rp, "?")
		p = fs[0]
		if len(fs) > 1 {
			q = fs[1]
		}

		if p[0] == '/' {
			u.Path = p
		} else {
			u.Path += p
		}
		u.RawQuery = q
	}
}

func getServiceDescriptions(rootURL string, device device, localIP net.IP, client *http.Client) ([]Service, error) {
	var result []Service

	if device.DeviceType == URN_InterntGatewayDevice1 {
		descriptions, err := getIGDServices(rootURL, localIP, client, device,
			URN_WANDevice1,
			URN_WANConnectionDevice1,
			URN_WANConnections1,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, descriptions...)
	} else if device.DeviceType == URN_InternetGatewayDevice2 {
		descriptions, err := getIGDServices(rootURL, localIP, client, device,
			URN_WANDevice2,
			URN_WANConnectionDevice2,
			URN_WANConnections2,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, descriptions...)
	} else {
		return result, errors.New("[" + rootURL + "] Malformed root device description: not an InternetGatewayDevice.")
	}

	if len(result) < 1 {
		return result, errors.New("[" + rootURL + "] Malformed device description: no compatible service descriptions found.")
	}

	return result, nil
}

func getIGDServices(rootURL string, localIP net.IP, client *http.Client, device device, wanDeviceURN string, wanConnectionURN string, URNs []string) ([]Service, error) {
	var result []Service

	devices := getChildDevices(device, wanDeviceURN)

	if len(devices) < 1 {
		return nil, fmt.Errorf("malformed InternetGatewayDevice description: no WANDevices specified")
	}

	for _, device := range devices {
		connections := getChildDevices(device, wanConnectionURN)

		for _, connection := range connections {
			for _, URN := range URNs {
				for _, service := range getChildServices(connection, URN) {
					if 0 < len(service.ControlURL) {
						u, _ := url.Parse(rootURL)
						replaceRawPath(u, service.ControlURL)
						service := Service{
							ID:      service.ID,
							URL:     u.String(),
							URN:     service.Type,
							LocalIP: localIP,
							Client:  client,
						}
						result = append(result, service)
					}
				}
			}
		}
	}

	return result, nil
}

func localIP(url *url.URL) (net.IP, error) {
	conn, err := net.DialTimeout("tcp", url.Host, time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localIPAddress, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return nil, err
	}

	return net.ParseIP(localIPAddress), nil
}

func getChildServices(d device, serviceType string) []service {
	var result []service
	for _, service := range d.Services {
		if service.Type == serviceType {
			result = append(result, service)
		}
	}
	return result
}

func getChildDevices(d device, deviceType string) []device {
	var result []device
	for _, dev := range d.Devices {
		if dev.DeviceType == deviceType {
			result = append(result, dev)
		}
	}
	return result
}
