package receive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pion/webrtc/v3"
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
	cobraCommand       *cobra.Command
	fileTransferConfig *file.WriteTransferConfig
	webrtcConfig       *webrtc.Configuration
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "receive",
		Short: "Receive a message from a WebRTC client",
		Long:  "Receive a message from a WebRTC client",
		RunE:  c.receive,
	}

	flags := c.cobraCommand.Flags()
	flags.StringP("offer-file", "O", "", "Path to file containing the SDP offer.")
	flags.StringP("answer-file", "A", "", "Path to file which the SDP answer should be written to.")

	return c.cobraCommand
}

func (c *Cmd) receive(cmd *cobra.Command, args []string) error {
	var (
		ctx = cmd.Context()

		flags                  = cmd.Flags()
		offerFilePath, _       = flags.GetString("offer-file")
		answerFilePath, _      = flags.GetString("answer-file")
		webRTCSignallingDir, _ = flags.GetString("webrtc-signalling-dir")
		webRTCSignallingURL, _ = flags.GetString("webrtc-signalling-server-url")
	)

	output.InvocationInfo(ctx, cmd, args)

	if err := c.configureWebRTC(flags); err != nil {
		return err
	}

	transport, err := client.NewTransport(c.webrtcConfig)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	var signaller signallers.ClientSignaller
	if webRTCSignallingDir != "" && webRTCSignallingURL != "" {
		return fmt.Errorf("cannot use both --webrtc-signalling-dir and --webrtc-signalling-server-url")
	}
	if webRTCSignallingDir == "" && webRTCSignallingURL == "" {
		return fmt.Errorf("must specify either --webrtc-signalling-dir or --webrtc-signalling-server-url")
	}

	if webRTCSignallingDir != "" {
		signaller = signallers.NewFileClientSignaller(offerFilePath, answerFilePath)
	} else {
		signaller = signallers.NewServerClientSignaller(webRTCSignallingURL, nil)
	}

	go func() {
		if err := signaller.Start(ctx, transport); err != nil {
			log.Printf("signaller error: %v", err)
		}
	}()
	defer signaller.Shutdown()

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

	req, err := http.NewRequest(http.MethodGet, "http://"+host, nil)
	if err != nil {
		return err
	}
	req.Close = true
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
		return err
	}

	wts, err := c.fileTransferConfig.NewWriteTransferSession(ctx, "", "")
	if err != nil {
		return err
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
