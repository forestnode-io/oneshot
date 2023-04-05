package receive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p/client/discovery"
	"github.com/raphaelreyna/oneshot/v2/pkg/configuration"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/file"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/client"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp/signallers"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func New(config *configuration.Root) *Cmd {
	return &Cmd{
		config: config,
	}
}

type Cmd struct {
	cobraCommand       *cobra.Command
	fileTransferConfig *file.WriteTransferConfig
	webrtcConfig       *webrtc.Configuration
	config             *configuration.Root
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "receive [file]",
		Short: "Receive from a sending oneshot instance over p2p",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			config := c.config.Subcommands.P2P.Client.Receive
			config.MergeFlags()
			return config.Validate()
		},
		RunE: c.receive,
	}

	c.config.Subcommands.P2P.Client.Receive.SetFlags(c.cobraCommand, c.cobraCommand.Flags())

	return c.cobraCommand
}

func (c *Cmd) receive(cmd *cobra.Command, args []string) error {
	var (
		ctx = cmd.Context()
		log = zerolog.Ctx(ctx)

		config    = c.config.Subcommands.P2P.Client.Receive
		p2pConfig = c.config.NATTraversal.P2P
		dsConfig  = c.config.NATTraversal.DiscoveryServer
		baConfig  = c.config.BasicAuth
	)

	output.InvocationInfo(ctx, cmd, args)

	err := c.configureWebRTC()
	if err != nil {
		return err
	}

	var (
		signaller signallers.ClientSignaller
		transport *client.Transport
		bat       string
	)
	if p2pConfig.DiscoveryDir != "" {
		transport, err = client.NewTransport(c.webrtcConfig)
		if err != nil {
			return fmt.Errorf("failed to create transport: %w", err)
		}
		signaller, bat, err = signallers.NewFileClientSignaller(config.OfferFile, config.AnswerFile)
	} else {
		corr, err := discovery.NegotiateOfferRequest(ctx, dsConfig.URL, baConfig.Username, baConfig.Password, http.DefaultClient)
		if err != nil {
			return fmt.Errorf("failed to negotiate offer request: %w", err)
		}
		transport, err = client.NewTransport(corr.RTCConfiguration)
		if err != nil {
			return fmt.Errorf("failed to create transport: %w", err)
		}
		signaller, bat, err = signallers.NewServerClientSignaller(dsConfig.URL, corr.SessionID, corr.RTCSessionDescription, nil)
	}
	if err != nil {
		return fmt.Errorf("failed to create signaller: %w", err)
	}

	go func() {
		if err := signaller.Start(ctx, transport); err != nil {
			log.Printf("signaller error: %v", err)
		}
	}()
	defer signaller.Shutdown()

	log.Debug().Msg("waiting for connection to oneshot server to be established")

	if err = transport.WaitForConnectionEstablished(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			log.Printf("... connection not established: %v", err)
			return nil
		}
	}

	log.Debug().Msg("connection to oneshot server established")

	preferredAddress, preferredPort := oneshotnet.PreferNonPrivateIP(transport.PeerAddresses())
	host := "http://localhost:8080"
	if preferredAddress != "" {
		host = net.JoinHostPort(preferredAddress, preferredPort)
	}

	req, err := http.NewRequest(http.MethodGet, "http://"+host, nil)
	if err != nil {
		return err
	}
	req.Close = true
	if bat != "" {
		req.Header.Set("X-HTTPOverWebRTC-Authorization", bat)
	}
	if preferredAddress != "" {
		req.RemoteAddr = host
	}

	events.Raise(ctx, output.NewHTTPRequest(req))

	httpClient := http.Client{
		Transport: transport,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to receive file: %s", resp.Status)
	}

	log.Debug().
		Int("status", resp.StatusCode).
		Interface("headers", resp.Header).
		Msg("received response from oneshot server")

	cl := int64(0)
	clString := resp.Header.Get("Content-Length")
	if clString == "" {
		cl, err = strconv.ParseInt(clString, 10, 64)
		if err == nil {
			cl = 0
		}
	}

	var location string
	if 0 < len(args) {
		location = args[0]
	}
	c.fileTransferConfig, err = file.NewWriteTransferConfig(ctx, location)
	if err != nil {
		log.Error().Err(err).
			Msg("failed to create file transfer config")

		return fmt.Errorf("failed to create file transfer config: %w", err)
	}

	wts, err := c.fileTransferConfig.NewWriteTransferSession(ctx, "", "")
	if err != nil {
		log.Error().Err(err).
			Msg("failed to create write transfer session")

		return fmt.Errorf("failed to create write transfer session: %w", err)
	}
	defer wts.Close()

	cancelProgDisp := output.DisplayProgress(
		ctx,
		&wts.Progress,
		125*time.Millisecond,
		req.RemoteAddr,
		cl,
	)
	defer cancelProgDisp()

	body, buf := output.NewBufferedReader(ctx, resp.Body)
	fileReport := events.File{
		Size:              cl,
		TransferStartTime: time.Now(),
	}

	n, err := io.Copy(wts, body)
	if err != nil {
		log.Error().Err(err).
			Msg("failed to copy response body to file")

		return fmt.Errorf("failed to copy response body to file after %d bytes: %w", n, err)
	}
	fileReport.TransferEndTime = time.Now()
	if buf != nil {
		fileReport.TransferSize = int64(buf.Len())
		fileReport.Content = buf.Bytes()
	}

	events.Raise(ctx, &fileReport)
	events.Success(ctx)
	events.Stop(ctx)

	return err
}

func (c *Cmd) configureWebRTC() error {
	var (
		config = c.config.NATTraversal.P2P.WebRTCConfiguration
		err    error
	)
	c.webrtcConfig, err = config.WebRTCConfiguration()
	if err != nil {
		return fmt.Errorf("unable to configure webrtc: %w", err)
	}

	return nil
}
