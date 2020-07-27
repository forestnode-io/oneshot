package cmd

import (
	"github.com/grandcat/zeroconf"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"os"
	"strconv"
)

func (a *App) MDNS(version string, srvr *server.Server) error {
	// If we are using mdns, the zeroconf server needs to be started up,
	// and the human readable address needs to be prepended to the list of ip addresses.
	conf := a.conf
	if conf.Mdns {
		portN, err := strconv.ParseInt(conf.Port, 10, 32)
		if err != nil {
			return err
		}

		mdnsSrvr, err := zeroconf.Register(
			"oneshot",
			"_http._tcp",
			"local.",
			int(portN),
			[]string{"version=" + version},
			nil,
		)
		defer mdnsSrvr.Shutdown()
		if err != nil {
			return err
		}

		host, err := os.Hostname()
		if err != nil {
			return err
		}

		srvr.HostAddresses = append(
			[]string{host + ".local" + ":" + conf.Port},
			srvr.HostAddresses...,
		)
	}
	return nil
}
