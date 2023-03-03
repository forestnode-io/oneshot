package upnpigd

import "net"

type service struct {
	ID         string `xml:"serviceId"`
	Type       string `xml:"serviceType"`
	ControlURL string `xml:"controlURL"`
}

type device struct {
	DeviceType   string    `xml:"deviceType"`
	FriendlyName string    `xml:"friendlyName"`
	Devices      []device  `xml:"deviceList>device"`
	Services     []service `xml:"serviceList>service"`
}

type root struct {
	Device device `xml:"device"`
}

const (
	URN_InterntGatewayDevice1 = "urn:schemas-upnp-org:device:InternetGatewayDevice:1"
	URN_WANDevice1            = "urn:schemas-upnp-org:device:WANDevice:1"
	URN_WANConnectionDevice1  = "urn:schemas-upnp-org:device:WANConnectionDevice:1"
	URN_WANIPConnection1      = "urn:schemas-upnp-org:service:WANIPConnection:1"
	URN_WANPPPConnection1     = "urn:schemas-upnp-org:service:WANPPPConnection:1"

	URN_InternetGatewayDevice2 = "urn:schemas-upnp-org:device:InternetGatewayDevice:2"
	URN_WANDevice2             = "urn:schemas-upnp-org:device:WANDevice:2"
	URN_WANConnectionDevice2   = "urn:schemas-upnp-org:device:WANConnectionDevice:2"
	URN_WANIPConnection2       = "urn:schemas-upnp-org:service:WANIPConnection:2"
	URN_WANPPPConnection2      = "urn:schemas-upnp-org:service:WANPPPConnection:2"
)

var (
	URN_InternetGatewayDevices = []string{
		URN_InterntGatewayDevice1,
		URN_InternetGatewayDevice2,
	}
	URN_WANConnections1 = []string{
		URN_WANIPConnection1,
		URN_WANPPPConnection1,
	}
	URN_WANConnections2 = []string{
		URN_WANIPConnection2,
		URN_WANPPPConnection2,
	}
)

var MulticastADDR = net.UDPAddr{
	IP:   net.IPv4(239, 255, 255, 250),
	Port: 1900,
}
