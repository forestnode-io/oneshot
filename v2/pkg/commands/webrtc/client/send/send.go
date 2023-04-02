package send

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/client/discovery"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/file"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
	oneshotwebrtc "github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/client"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp/signallers"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func New() *Cmd {
	return &Cmd{}
}

type Cmd struct {
	cobraCommand  *cobra.Command
	archiveMethod archiveFlag
	webrtcConfig  *webrtc.Configuration
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "send [file|dir]",
		Short: "Send a message to a WebRTC client",
		Long:  "Send a message to a WebRTC client",
		RunE:  c.send,
	}

	flags := c.cobraCommand.Flags()
	flags.StringP("name", "n", "", "Name of file presented to the server.")
	flags.StringP("offer-file", "O", "", "Path to file containing the SDP offer.")
	flags.StringP("answer-file", "A", "", "Path to file which the SDP answer should be written to.")
	flags.VarP(&c.archiveMethod, "archive-method", "a", `Which archive method to use when sending directories.
Recognized values are "zip", "tar" and "tar.gz".`)
	if runtime.GOOS == "windows" {
		flags.Lookup("archive-method").DefValue = "zip"
	} else {
		flags.Lookup("archive-method").DefValue = "tar.gz"
	}

	return c.cobraCommand
}

func (c *Cmd) send(cmd *cobra.Command, args []string) error {
	var (
		ctx   = cmd.Context()
		paths = args

		flags                  = cmd.Flags()
		fileName, _            = flags.GetString("name")
		offerFilePath, _       = flags.GetString("offer-file")
		answerFilePath, _      = flags.GetString("answer-file")
		webRTCSignallingDir, _ = flags.GetString("webrtc-signalling-dir")
		webRTCSignallingURL, _ = flags.GetString("webrtc-signalling-server-url")

		username, _ = flags.GetString("username")
		password, _ = flags.GetString("password")
	)

	output.InvocationInfo(ctx, cmd, args)

	if len(paths) == 1 && fileName == "" {
		fileName = filepath.Base(paths[0])
	}

	if fileName == "" {
		fileName = namesgenerator.GetRandomName(0)
	}

	err := c.configureWebRTC(flags)
	if err != nil {
		return err
	}

	if webRTCSignallingDir != "" && webRTCSignallingURL != "" {
		return fmt.Errorf("cannot use both --webrtc-signalling-dir and --webrtc-signalling-server-url")
	}
	if webRTCSignallingDir == "" && webRTCSignallingURL == "" {
		return fmt.Errorf("must specify either --webrtc-signalling-dir or --webrtc-signalling-server-url")
	}

	var (
		signaller signallers.ClientSignaller
		transport *client.Transport
		bat       string
	)
	if webRTCSignallingDir != "" {
		transport, err = client.NewTransport(c.webrtcConfig)
		if err != nil {
			return fmt.Errorf("failed to create transport: %w", err)
		}
		signaller, bat, err = signallers.NewFileClientSignaller(offerFilePath, answerFilePath)
	} else {
		corr, err := discovery.NegotiateOfferRequest(ctx, webRTCSignallingURL, username, password, http.DefaultClient)
		if err != nil {
			return fmt.Errorf("failed to negotiate offer request: %w", err)
		}
		transport, err = client.NewTransport(corr.RTCConfiguration)
		if err != nil {
			return fmt.Errorf("failed to create transport: %w", err)
		}
		signaller, bat, err = signallers.NewServerClientSignaller(webRTCSignallingURL, corr.SessionID, corr.RTCSessionDescription, nil)
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

	archiveMethod := string(c.archiveMethod)
	if archiveMethod == "" {
		archiveMethod = flags.Lookup("archive-method").DefValue
	}

	rtc, err := file.NewReadTransferConfig(archiveMethod, args...)
	if err != nil {
		return err
	}

	if file.IsArchive(rtc) {
		fileName += "." + archiveMethod
	}

	header := http.Header{}
	if bat != "" {
		header.Set("X-HTTPOverWebRTC-Authorization", bat)
	}
	rts, err := rtc.NewReaderTransferSession(ctx)
	if err != nil {
		return err
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

	log.Println("waiting for connection to oneshot server to be established...")
	if err = transport.WaitForConnectionEstablished(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			log.Printf("... connection not established: %v", err)
			return nil
		}
	}
	log.Println("... connection established")

	preferredAddress, preferredPort := oneshotnet.PreferNonPrivateIP(transport.PeerAddresses())

	host := "http://localhost:8080"
	if preferredAddress != "" {
		host = net.JoinHostPort(preferredAddress, preferredPort)
	}

	req, err := http.NewRequest(http.MethodPost, "http://"+host, body)
	if err != nil {
		return err
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
		return err
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

type archiveFlag string

func (a *archiveFlag) String() string {
	return string(*a)
}

func (a *archiveFlag) Set(value string) error {
	switch value {
	case "zip", "tar", "tar.gz":
		*a = archiveFlag(value)
		return nil
	default:
		return fmt.Errorf(`invalid archive method %q, must be "zip", "tar" or "tar.gz`, value)
	}
}

func (a archiveFlag) Type() string {
	return "string"
}

func (c *Cmd) configureWebRTC(flags *pflag.FlagSet) error {
	path, _ := flags.GetString("webrtc-config-file")
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read webrtc config file: %w", err)
	}

	config := oneshotwebrtc.Configuration{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("unable to parse webrtc config file: %w", err)
	}

	c.webrtcConfig, err = config.WebRTCConfiguration()
	if err != nil {
		return fmt.Errorf("unable to configure webrtc: %w", err)
	}

	return nil
}
