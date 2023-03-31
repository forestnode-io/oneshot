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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

func (r *rootCommand) listenWebRTC(ctx context.Context, addrChan chan<- string, ba *messages.BasicAuth) error {
	r.wg.Add(1)
	defer r.wg.Done()

	var (
		flags                  = r.Flags()
		useWebRTC, _           = flags.GetBool("p2p")
		webRTCSignallingDir, _ = flags.GetString("p2p-discovery-dir")
		webRTCSignallingURL, _ = flags.GetString("p2p-discovery-server-url")
		webrtcOnly, _          = flags.GetBool("p2p-only")
	)

	if !useWebRTC &&
		!webrtcOnly &&
		webRTCSignallingDir == "" &&
		webRTCSignallingURL == "" {
		close(addrChan)
		return nil
	}

	err := r.configureWebRTC(flags)
	if err != nil {
		return fmt.Errorf("failed to configure p2p: %w", err)
	}

	signaller, err := getSignaller(ctx, flags, r.webrtcConfig, ba)
	if err != nil {
		return fmt.Errorf("failed to get p2p signaller: %w", err)
	}
	defer signaller.Shutdown()

	a := server.NewServer(r.webrtcConfig, http.HandlerFunc(r.server.ServeHTTP))
	defer a.Wait()

	log.Println("starting p2p discovery mechanism")
	if err := signaller.Start(ctx, a, addrChan); err != nil {
		log.Fatal(err)
	}

	log.Println("p2p discovery mechanism started")

	return nil
}

func getSignaller(ctx context.Context, flags *pflag.FlagSet, config *webrtc.Configuration, ba *messages.BasicAuth) (signallers.ServerSignaller, error) {
	var (
		webRTCSignallingURL, _ = flags.GetString("p2p-discovery-server-url")
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
		return newServerServerSignaller(flags, ba, kacp), nil
	}

	return nil, fmt.Errorf("no p2p discovery mechanism specified")
}

func newServerServerSignaller(flags *pflag.FlagSet, ba *messages.BasicAuth, kacp keepalive.ClientParameters) signallers.ServerSignaller {
	var (
		url, _               = flags.GetString("p2p-discovery-server-url")
		id, _                = flags.GetString("p2p-discovery-server-id")
		assignURL, _         = flags.GetString("p2p-discovery-server-request-url")
		assignRequiredURL, _ = flags.GetString("p2p-discovery-server-required-url")
	)

	urlRequired := false
	if assignRequiredURL != "" {
		assignURL = assignRequiredURL
		urlRequired = true
	}

	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(kacp),
	}
	conf := signallers.ServerServerSignallerConfig{
		ID:                  id,
		SignallingServerURL: url,

		URL:         assignURL,
		URLRequired: urlRequired,

		BasicAuth: ba,
	}
	if useInsecure, _ := flags.GetBool("p2p-discovery-server-insecure"); useInsecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	}
	conf.GRPCOpts = opts

	return signallers.NewServerServerSignaller(&conf)
}
