package network

import (
	"net"
	"strings"
)

func HostAddresses() ([]string, error) {
	ifaceAddrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	hostAddrs := []string{}
	// run the loop backwards so 127.0.0.1 ends up at the bottom of the list
	for idx := len(ifaceAddrs) - 1; 0 <= idx; idx-- {
		addr := ifaceAddrs[idx].String()
		if strings.Contains(addr, "::") {
			continue
		}

		parts := strings.Split(addr, "/")
		ip := net.ParseIP(parts[0])
		if ip == nil {
			continue
		}

		// Filter out IPv6 address
		if ip.To4() == nil {
			continue
		}

		hostAddrs = append(hostAddrs, ip.String())
	}

	return hostAddrs, nil
}
