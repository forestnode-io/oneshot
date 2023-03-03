package upnpigd

type service struct {
	ID         string `xml:"serviceId"`
	Type       string `xml:"serviceType"`
	ControlURL string `xml:"controlURL"`
}

type device struct {
	DeviceType   string `xml:"deviceType"`
	FriendlyName string `xml:"friendlyName"`
	Devices      []any  `xml:"deviceList>device"`
	Services     []any  `xml:"serviceList>service"`
}

type root struct {
	Device any `xml:"device"`
}
