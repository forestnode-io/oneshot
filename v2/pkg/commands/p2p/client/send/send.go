package send

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p/client/discovery"
	"github.com/raphaelreyna/oneshot/v2/pkg/configuration"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/file"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/client"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp/signallers"
	oneshotos "github.com/raphaelreyna/oneshot/v2/pkg/os"
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
	cobraCommand *cobra.Command
	webrtcConfig *webrtc.Configuration
	config       *configuration.Root
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "send [file|dir]",
		Short: "Send to a receiving oneshot instance over p2p",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			config := c.config.Subcommands.P2P.Client.Send
			config.MergeFlags()
			if err := config.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}
			return nil
		},
		RunE: c.send,
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)

	c.config.Subcommands.P2P.Client.Send.SetFlags(c.cobraCommand, c.cobraCommand.LocalFlags())

	return c.cobraCommand
}

func (c *Cmd) send(cmd *cobra.Command, args []string) error {
	var (
		ctx   = cmd.Context()
		log   = zerolog.Ctx(ctx)
		paths = args

		config    = c.config.Subcommands.P2P.Client.Send
		p2pConfig = c.config.NATTraversal.P2P
		dsConfig  = c.config.NATTraversal.DiscoveryServer
		baConfig  = c.config.BasicAuth

		fileName            = config.Name
		webRTCSignallingDir = p2pConfig.DiscoveryDir
		webRTCSignallingURL = dsConfig.URL
	)

	output.InvocationInfo(ctx, cmd, args)

	if len(paths) == 1 && fileName == "" {
		fileName = filepath.Base(paths[0])
	}

	if fileName == "" {
		fileName = namesgenerator.GetRandomName(0)
	}

	err := c.configureWebRTC()
	if err != nil {
		log.Error().Err(err).
			Msg("failed to configure webrtc")

		return fmt.Errorf("failed to configure webrtc: %w", err)
	}

	var (
		signaller signallers.ClientSignaller
		transport *client.Transport
		bat       string
	)
	if webRTCSignallingDir != "" {
		transport, err = client.NewTransport(c.webrtcConfig)
		if err != nil {
			log.Error().Err(err).
				Msg("failed to create transport")

			return fmt.Errorf("failed to create transport: %w", err)
		}

		foundPair := false
		offerFilePath := filepath.Join(webRTCSignallingDir, "offer")
		answerFilePath := filepath.Join(webRTCSignallingDir, "answer")
		_, err := os.Stat(offerFilePath)
		if err == nil {
			_, err = os.Stat(answerFilePath)
			if err == nil {
				foundPair = true
			}
		}
		if !foundPair {
			dirContents, err := oneshotos.ReadDirSorted(webRTCSignallingDir, true)
			if err != nil {
				log.Error().Err(err).
					Msg("failed to read dir")

				return fmt.Errorf("failed to read dir: %w", err)
			}
			latestDir := dirContents[len(dirContents)-1].Name()

			ofp := filepath.Join(webRTCSignallingDir, latestDir, "offer")
			afp := filepath.Join(webRTCSignallingDir, latestDir, "answer")
			_, err = os.Stat(ofp)
			if err == nil {
				_, err = os.Stat(afp)
				if err == nil {
					offerFilePath = ofp
					answerFilePath = afp
					foundPair = true
				}
			}
		}
		if !foundPair {
			log.Error().
				Msg("no offer/answer pair found")

			return fmt.Errorf("no offer/answer pair found")
		}

		signaller, bat, err = signallers.NewFileClientSignaller(offerFilePath, answerFilePath)
	} else {
		corr, err := discovery.NegotiateOfferRequest(ctx, webRTCSignallingURL, baConfig.Username, baConfig.Password, http.DefaultClient)
		if err != nil {
			log.Error().Err(err).
				Msg("failed to negotiate offer request")

			return fmt.Errorf("failed to negotiate offer request: %w", err)
		}
		transport, err = client.NewTransport(corr.RTCConfiguration)
		if err != nil {
			log.Error().Err(err).
				Msg("failed to create transport")

			return fmt.Errorf("failed to create transport: %w", err)
		}
		signaller, bat, err = signallers.NewServerClientSignaller(webRTCSignallingURL, corr.SessionID, corr.RTCSessionDescription, nil)
	}
	if err != nil {
		log.Error().Err(err).
			Msg("failed to create signaller")

		return fmt.Errorf("failed to create signaller: %w", err)
	}

	go func() {
		if err := signaller.Start(ctx, transport); err != nil {
			log.Error().Err(err).
				Msg("failed to start signaller")
		}
	}()
	defer signaller.Shutdown()

	rtc, err := file.NewReadTransferConfig(config.ArchiveMethod.String(), args...)
	if err != nil {
		log.Error().Err(err).
			Msg("failed to create read transfer config")

		return fmt.Errorf("failed to create read transfer config: %w", err)
	}

	if file.IsArchive(rtc) {
		fileName += "." + config.ArchiveMethod.String()
	}

	header := http.Header{}
	if bat != "" {
		header.Set("X-HTTPOverWebRTC-Authorization", bat)
	}
	rts, err := rtc.NewReaderTransferSession(ctx)
	if err != nil {
		log.Error().Err(err).
			Msg("failed to create reader transfer session")

		return fmt.Errorf("failed to create reader transfer session: %w", err)
	}
	defer rts.Close()
	size, err := rts.Size()
	if err == nil {
		cl := fmt.Sprintf("%d", size)
		header["Content-Length"] = []string{cl}
	}

	body, buf := output.NewBufferedReader(ctx, rts)
	fileReport := events.File{
		Size:              int64(size),
		TransferStartTime: time.Now(),
	}

	log.Debug().Msg("waiting for connection to oneshot server to be established")
	if err = transport.WaitForConnectionEstablished(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			log.Error().Err(err).
				Msg("failed to establish connection to oneshot server")

			return nil
		}
	}
	log.Debug().Msg("connection established")

	preferredAddress, preferredPort := oneshotnet.PreferNonPrivateIP(transport.PeerAddresses())

	// We need to provide a host header to the request but
	// it doesnt influence anything in this case since the custom webrtc transport ignores it.
	host := "http://localhost:8080"
	if preferredAddress != "" {
		host = net.JoinHostPort(preferredAddress, preferredPort)
	}

	req, err := http.NewRequest(http.MethodPost, "http://"+host, body)
	if err != nil {
		log.Error().Err(err).
			Msg("failed to create request")

		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header = header
	req.Close = true
	if preferredAddress != "" {
		req.RemoteAddr = host
	}

	httpClient := http.Client{
		Transport: transport,
	}

	events.Raise(ctx, output.NewHTTPRequest(req))
	cancelProgDisp := output.DisplayProgress(
		cmd.Context(),
		&rts.Progress,
		125*time.Millisecond,
		req.RemoteAddr,
		size,
	)
	defer cancelProgDisp()

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).
			Msg("failed to send file")

		return fmt.Errorf("failed to send file: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send file: %s", resp.Status)
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
