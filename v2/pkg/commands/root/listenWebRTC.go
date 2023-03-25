package root

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp/signallers"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/server"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
)

func (r *rootCommand) listenWebRTC(ctx context.Context, ba *messages.BasicAuth) error {
	r.wg.Add(1)
	defer r.wg.Done()

	var (
		flags                  = r.Flags()
		webrtc, _              = flags.GetBool("webrtc")
		webRTCSignallingDir, _ = flags.GetString("webrtc-signalling-dir")
		webRTCSignallingURL, _ = flags.GetString("webrtc-signalling-server-url")
		webrtcOnly, _          = flags.GetBool("webrtc-only")
	)

	if !webrtc &&
		!webrtcOnly &&
		webRTCSignallingDir == "" &&
		webRTCSignallingURL == "" {
		return nil
	}

	err := r.configureWebRTC(flags)
	if err != nil {
		return fmt.Errorf("failed to configure WebRTC: %w", err)
	}

	signaller, err := getSignaller(ctx, flags, ba)
	if err != nil {
		return fmt.Errorf("failed to get WebRTC signaller: %w", err)
	}
	defer signaller.Shutdown()

	a := server.NewServer(r.webrtcConfig, http.HandlerFunc(r.server.ServeHTTP))
	defer a.Wait()

	log.Println("starting WebRTC signalling mechanism")
	if err := signaller.Start(ctx, a); err != nil {
		log.Fatal(err)
	}

	log.Println("WebRTC signalling mechanism started")

	return nil
}

func getSignaller(ctx context.Context, flags *pflag.FlagSet, ba *messages.BasicAuth) (signallers.ServerSignaller, error) {
	var (
		webRTCSignallingURL, _         = flags.GetString("webrtc-signalling-server-url")
		webRTCSignallingDir, _         = flags.GetString("webrtc-signalling-dir")
		webRTCSignallingID, _          = flags.GetString("webrtc-signalling-server-id")
		webRTCSignallingRequestURL, _  = flags.GetString("webrtc-signalling-server-request-url")
		webRTCSignallingRequiredURL, _ = flags.GetString("webrtc-signalling-server-required-url")
	)

	webrtcClientURL := webRTCSignallingRequestURL
	webrtcClientURLRequired := false
	if webRTCSignallingRequiredURL != "" {
		webrtcClientURL = webRTCSignallingRequiredURL
		webrtcClientURLRequired = true
	}

	if output.IsTTYForContentOnly(ctx) {
		if webRTCSignallingDir == "" && webRTCSignallingURL == "" {
			return nil, fmt.Errorf("signalling directory (--webrtc-signalling-dir) or signalling server url (--webrtc-signalling-server-url) must be set when serving from stdin or to stdout")
		}
		if webRTCSignallingURL != "" {
			conf := signallers.ServerServerSignallerConfig{
				ID:                  webRTCSignallingID,
				SignallingServerURL: webRTCSignallingURL,
				GRPCOpts: []grpc.DialOption{
					grpc.WithInsecure(),
					grpc.WithBlock(),
					grpc.WithTimeout(5 * time.Second),
				},

				URL:         webrtcClientURL,
				URLRequired: webrtcClientURLRequired,

				BasicAuth: ba,
			}
			return signallers.NewServerServerSignaller(&conf), nil
		}

		return signallers.NewFileServerSignaller(webRTCSignallingDir), nil
	} else if webRTCSignallingDir != "" {
		return signallers.NewFileServerSignaller(webRTCSignallingDir), nil
	} else if webRTCSignallingURL != "" {
		conf := signallers.ServerServerSignallerConfig{
			ID:                  webRTCSignallingID,
			SignallingServerURL: webRTCSignallingURL,
			GRPCOpts: []grpc.DialOption{
				grpc.WithInsecure(),
				grpc.WithBlock(),
				grpc.WithTimeout(5 * time.Second),
			},

			URL:         webrtcClientURL,
			URLRequired: webrtcClientURLRequired,

			BasicAuth: ba,
		}
		return signallers.NewServerServerSignaller(&conf), nil
	}

	return signallers.NewTTYServerSignaller(), nil
}
