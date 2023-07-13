package root

import (
	"context"
	"fmt"
	"net/http"

	"github.com/oneshot-uno/oneshot/v2/pkg/configuration"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/sdp/signallers"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/server"
	"github.com/rs/zerolog"
)

func (r *rootCommand) listenWebRTC(ctx context.Context, bat, portMapAddr string) error {
	r.wg.Add(1)
	defer r.wg.Done()

	var (
		log = zerolog.Ctx(ctx)
	)

	err := r.configureWebRTC()
	if err != nil {
		return fmt.Errorf("failed to configure p2p: %w", err)
	}

	signaller, err := getSignaller(ctx, r.config, portMapAddr)
	if err != nil {
		return fmt.Errorf("failed to get p2p signaller: %w", err)
	}
	defer signaller.Shutdown()

	// create a webrtc server with the same handler as the http server
	a := server.NewServer(r.webrtcConfig, bat, http.HandlerFunc(r.server.ServeHTTP))
	defer a.Wait()

	log.Info().Msg("starting p2p discovery mechanism")
	if err := signaller.Start(ctx, a); err != nil {
		return fmt.Errorf("failed to start p2p discovery mechanism: %w", err)
	}

	return nil
}

func getSignaller(ctx context.Context, config *configuration.Root, portMapAddr string) (signallers.ServerSignaller, error) {
	var (
		dsConf              = config.Discovery
		p2pConf             = config.NATTraversal.P2P
		webRTCSignallingURL = dsConf.Host
		webRTCSignallingDir = p2pConf.DiscoveryDir
	)

	if webRTCSignallingDir != "" {
		if config == nil {
			return nil, fmt.Errorf("nil p2p configuration")
		}
		iwc, err := p2pConf.ParseConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to parse p2p configuration: %w", err)
		}
		wc, err := iwc.WebRTCConfiguration()
		if err != nil {
			return nil, fmt.Errorf("failed to get WebRTC configuration: %w", err)
		}
		return signallers.NewFileServerSignaller(webRTCSignallingDir, wc), nil
	} else if webRTCSignallingURL != "" {
		return signallers.NewServerServerSignaller(), nil
	}

	return nil, fmt.Errorf("no p2p discovery mechanism specified")
}

func (r *rootCommand) configureWebRTC() error {
	conf := r.config.NATTraversal.P2P
	if len(conf.WebRTCConfiguration) == 0 {
		return nil
	}

	iwc, err := conf.ParseConfig()
	if err != nil {
		return fmt.Errorf("failed to parse p2p configuration: %w", err)
	}
	wc, err := iwc.WebRTCConfiguration()
	if err != nil {
		return fmt.Errorf("failed to get WebRTC configuration: %w", err)
	}
	r.webrtcConfig = wc

	return nil
}
