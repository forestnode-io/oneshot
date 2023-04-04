package root

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	upnpigd "github.com/raphaelreyna/oneshot/v2/pkg/net/upnp-igd"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
)

func (r *rootCommand) handlePortMap(ctx context.Context, flags *pflag.FlagSet) (string, func(), error) {
	var (
		log = zerolog.Ctx(ctx)

		externalPort, _        = flags.GetInt("external-port")
		portMappingDuration, _ = flags.GetDuration("port-mapping-duration")
		mapPort, _             = flags.GetBool("map-port")

		cancel = func() {}
	)

	userSetUPnPFlag := flags.Lookup("external-port").Changed || flags.Lookup("port-mapping-duration").Changed

	if !mapPort && !userSetUPnPFlag {
		return "", cancel, nil
	}

	port, _ := flags.GetInt("port")

	finishSpinning := output.DisplaySpinner(ctx,
		333*time.Millisecond,
		"negotiating port mapping",
		"negotiating port mapping ... done",
		[]string{".", "..", "...", ".."},
	)

	discoveryTimeout, _ := flags.GetDuration("upnp-discovery-timeout")
	devChan, err := upnpigd.Discover("oneshot", discoveryTimeout, http.DefaultClient)
	if err != nil {
		return "", cancel, fmt.Errorf("failed to discover UPnP IGD: %w", err)
	}

	devs := make([]*upnpigd.Device, 0)
	for dev := range devChan {
		devs = append(devs, dev)
	}

	if len(devs) == 0 {
		return "", cancel, errors.New("no UPnP IGD devices found")
	}

	dev := devs[0]
	if err := dev.AddPortMapping(ctx, "TCP", externalPort, port, "oneshot", portMappingDuration); err != nil {
		finishSpinning()
		return "", cancel, fmt.Errorf("failed to add port mapping: %w", err)
	}

	log.Info().
		Int("internal-port", port).
		Int("external-port", externalPort).
		Str("duration", portMappingDuration.String()).
		Msg("added port mapping")

	externalIP, err := dev.GetExternalIP(ctx)
	if err != nil {
		finishSpinning()
		return "", cancel, fmt.Errorf("failed to get external address: %w", err)
	}
	externalAddr := fmt.Sprintf("%s:%d", externalIP, externalPort)

	log.Info().
		Str("external-address", externalAddr).
		Msg("got external address")

	finishSpinning()

	// exit after the port mapping expires
	// TODO(raphaelreyna): this can be handled better.
	// Not sure about what happens when the port mapping expires while the server has
	// active connections.
	// Look into maybe refreshing the port mapping every so often while there are active
	// connections.
	t := time.AfterFunc(portMappingDuration, func() {
		events.Stop(ctx)
	})

	scheme := "http"
	if _, err := flags.GetString("tls-cert"); err != nil {
		scheme = "https"
	}
	externalAddr = fmt.Sprintf("%s://%s", scheme, externalAddr)

	return externalAddr, func() {
		t.Stop()
		if err := dev.DeletePortMapping(ctx, "TCP", externalPort); err != nil {
			log.Error().Err(err).
				Int("internal-port", port).
				Int("external-port", externalPort).
				Msg("failed to delete port mapping")
		}
		log.Info().
			Int("internal-port", port).
			Int("external-port", externalPort).
			Msg("deleted port mapping")

	}, nil
}
