package upnpigd

import (
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/oneshot-uno/oneshot/v2/pkg/log"
)

func Discover(userAgent string, timeout time.Duration, client *http.Client) (<-chan *Device, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var (
		log   = log.Logger()
		wg    sync.WaitGroup
		dc    = make(chan *Device, runtime.NumCPU())
		outDC = make(chan *Device, runtime.NumCPU())
	)
	wg.Add(1)

	// deduper
	go func() {
		seen := make(map[string]struct{})
		for d := range dc {
			if _, already := seen[d.uuid]; already {
				continue
			}
			seen[d.uuid] = struct{}{}
			outDC <- d
		}
		close(outDC)
	}()

	d := discoverer{
		MX:        timeout,
		Host:      &MulticastADDR,
		UserAgent: userAgent,
		Chan:      dc,
		Client:    client,
	}
	for _, iface := range ifaces {
		flags := iface.Flags
		if flags&net.FlagUp == 0 || flags&net.FlagMulticast == 0 {
			// these flags are 0 on windows so its ok, otherwise we skip
			if runtime.GOOS != "windows" {
				continue
			}
		}

		for _, st := range URN_InternetGatewayDevices {
			wg.Add(1)
			go func(iface net.Interface, st string) {
				defer wg.Done()
				if err := d.discover(iface, st); err != nil {
					log.Error().Err(err).
						Msg("unable to discover device")
				}
			}(iface, st)
		}
	}

	go func() {
		wg.Done()
		wg.Wait()
		close(dc)
	}()

	return outDC, nil
}

type discoverer struct {
	MX        time.Duration // timeout
	Host      *net.UDPAddr
	UserAgent string
	Chan      chan<- *Device
	Client    *http.Client
}

func (d *discoverer) discover(iface net.Interface, st string) error {
	log := log.Logger()
	tmplt := `M-SEARCH * HTTP/1.1
HOST: %s
ST: %s
MAN: "ssdp:discover"
MX: %d
USER-AGENT: %s

`

	payload := fmt.Sprintf(tmplt,
		d.Host,
		st,
		d.MX/time.Second,
		d.UserAgent,
	)
	payload = strings.ReplaceAll(payload, "\n", "\r\n") + "\r\n"

	udpAddr := net.UDPAddr{IP: d.Host.IP}
	conn, err := net.ListenMulticastUDP("udp4", &iface, &udpAddr)
	if err != nil {
		if strings.Contains(err.Error(), "no such network interface") ||
			strings.Contains(err.Error(), "i/o timeout") {
			return nil
		} else {
			return fmt.Errorf("unable to listen on %s: %v", iface.Name, err)
		}
	}
	defer conn.Close()

	if err = conn.SetDeadline(time.Now().Add(d.MX)); err != nil {
		return fmt.Errorf("unable to set deadline on %s: %v", iface.Name, err)
	}

	if _, err = conn.WriteTo([]byte(payload), d.Host); err != nil {
		return fmt.Errorf("unable to write to %s: %v", iface.Name, err)
	}

	buf := make([]byte, 65535)

	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			return fmt.Errorf("unable to read from %s: %v", iface.Name, err)
		}

		if n == 0 {
			continue
		}

		dev, err := NewDevice(d.Client, st, buf[:n])
		if err != nil {
			log.Error().Err(err).
				Msg("unable to parse device")
			continue
		}

		d.Chan <- dev
	}
}
