package root

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp/signallers"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/server"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/spf13/pflag"
	"google.golang.org/grpc/keepalive"
)

func (r *rootCommand) listenWebRTC(ctx context.Context, portMapAddr string, addrChan chan<- string, ba *messages.BasicAuth) error {
	r.wg.Add(1)
	defer r.wg.Done()

	var (
		flags = r.Flags()
	)

	err := r.configureWebRTC(flags)
	if err != nil {
		return fmt.Errorf("failed to configure p2p: %w", err)
	}

	signaller, err := getSignaller(ctx, flags, portMapAddr, r.webrtcConfig, ba)
	if err != nil {
		return fmt.Errorf("failed to get p2p signaller: %w", err)
	}
	defer signaller.Shutdown()

	// create a webrtc server with the same handler as the http server
	a := server.NewServer(r.webrtcConfig, http.HandlerFunc(r.server.ServeHTTP))
	defer a.Wait()

	log.Println("starting p2p discovery mechanism")
	if err := signaller.Start(ctx, a, addrChan); err != nil {
		return fmt.Errorf("failed to start p2p discovery mechanism: %w", err)
	}

	log.Println("p2p discovery mechanism started")

	return nil
}

func getSignaller(ctx context.Context, flags *pflag.FlagSet, portMapAddr string, config *webrtc.Configuration, ba *messages.BasicAuth) (signallers.ServerSignaller, error) {
	var (
		webRTCSignallingURL, _ = flags.GetString("discovery-server-url")
		webRTCSignallingDir, _ = flags.GetString("p2p-discovery-dir")

		kacp = keepalive.ClientParameters{
			Time:    6 * time.Second, // send pings every 6 seconds if there is no activity
			Timeout: time.Second,     // wait 1 second for ping ack before considering the connection dead
		}
	)

	if webRTCSignallingDir != "" {
		if config == nil {
			return nil, fmt.Errorf("nil p2p configuration")
		}
		return signallers.NewFileServerSignaller(webRTCSignallingDir, config), nil
	} else if webRTCSignallingURL != "" {
		return newServerServerSignaller(flags, portMapAddr, ba, kacp), nil
	}

	return nil, fmt.Errorf("no p2p discovery mechanism specified")
}

func newServerServerSignaller(flags *pflag.FlagSet, portMapAddr string, ba *messages.BasicAuth, kacp keepalive.ClientParameters) signallers.ServerSignaller {
	var (
		assignURL, _         = flags.GetString("p2p-discovery-server-request-url")
		assignRequiredURL, _ = flags.GetString("p2p-discovery-server-required-url")
	)

	urlRequired := false
	if assignRequiredURL != "" {
		assignURL = assignRequiredURL
		urlRequired = true
	}

	conf := signallers.ServerServerSignallerConfig{
		URL:         assignURL,
		URLRequired: urlRequired,
		PortMapAddr: "http://" + portMapAddr,
		BasicAuth:   ba,
	}

	return signallers.NewServerServerSignaller(&conf)
}
