package root

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/oneshot-uno/oneshot/v2/pkg/configuration"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/sdp/signallers"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/server"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/keepalive"
)

func (r *rootCommand) listenWebRTC(ctx context.Context, portMapAddr, bat string, addrChan chan<- string, ba *messages.BasicAuth) error {
	r.wg.Add(1)
	defer r.wg.Done()

	var (
		log = zerolog.Ctx(ctx)
	)

	err := r.configureWebRTC()
	if err != nil {
		return fmt.Errorf("failed to configure p2p: %w", err)
	}

	signaller, err := getSignaller(ctx, r.config, portMapAddr, ba)
	if err != nil {
		return fmt.Errorf("failed to get p2p signaller: %w", err)
	}
	defer signaller.Shutdown()

	// create a webrtc server with the same handler as the http server
	a := server.NewServer(r.webrtcConfig, bat, http.HandlerFunc(r.server.ServeHTTP))
	defer a.Wait()

	log.Info().Msg("starting p2p discovery mechanism")
	if err := signaller.Start(ctx, a, addrChan); err != nil {
		return fmt.Errorf("failed to start p2p discovery mechanism: %w", err)
	}

	return nil
}

func getSignaller(ctx context.Context, config *configuration.Root, portMapAddr string, ba *messages.BasicAuth) (signallers.ServerSignaller, error) {
	var (
		dsConf              = config.NATTraversal.DiscoveryServer
		p2pConf             = config.NATTraversal.P2P
		webRTCSignallingURL = dsConf.URL
		webRTCSignallingDir = p2pConf.DiscoveryDir

		kacp = keepalive.ClientParameters{
			Time:    6 * time.Second, // send pings every 6 seconds if there is no activity
			Timeout: time.Second,     // wait 1 second for ping ack before considering the connection dead
		}
	)

	if webRTCSignallingDir != "" {
		if config == nil {
			return nil, fmt.Errorf("nil p2p configuration")
		}
		wc, err := p2pConf.WebRTCConfiguration.WebRTCConfiguration()
		if err != nil {
			return nil, fmt.Errorf("failed to get WebRTC configuration: %w", err)
		}
		return signallers.NewFileServerSignaller(webRTCSignallingDir, wc), nil
	} else if webRTCSignallingURL != "" {
		return newServerServerSignaller(config, portMapAddr, ba, kacp), nil
	}

	return nil, fmt.Errorf("no p2p discovery mechanism specified")
}

func newServerServerSignaller(config *configuration.Root, portMapAddr string, ba *messages.BasicAuth, kacp keepalive.ClientParameters) signallers.ServerSignaller {
	var (
		dsConf            = config.NATTraversal.DiscoveryServer
		assignURL         = dsConf.PreferredURL
		assignRequiredURL = dsConf.RequiredURL
	)

	urlRequired := false
	if assignRequiredURL != "" {
		assignURL = assignRequiredURL
		urlRequired = true
	}

	if portMapAddr == "" && dsConf.OnlyRedirect {
		scheme := "http"
		if config.Server.TLSCert != "" {
			scheme = "https"
		}
		// we can omit the host, the discovery server will fill it in
		// with the host it receives the request from
		portMapAddr = fmt.Sprintf("%s://:%d", scheme, config.Server.Port)
	}

	conf := signallers.ServerServerSignallerConfig{
		URL:          assignURL,
		URLRequired:  urlRequired,
		PortMapAddr:  portMapAddr,
		BasicAuth:    ba,
		OnlyRedirect: dsConf.OnlyRedirect,
	}

	return signallers.NewServerServerSignaller(&conf)
}

func (r *rootCommand) configureWebRTC() error {
	conf := r.config.NATTraversal.P2P.WebRTCConfiguration
	if conf == nil {
		return nil
	}

	var err error
	r.webrtcConfig, err = conf.WebRTCConfiguration()
	if err != nil {
		return fmt.Errorf("unable to configure p2p: %w", err)
	}

	return nil
}
