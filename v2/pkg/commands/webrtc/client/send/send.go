package send

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/file"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/pkg/net"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/client"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/ice"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

		flags             = cmd.Flags()
		fileName, _       = flags.GetString("name")
		offerFilePath, _  = flags.GetString("offer-file")
		answerFilePath, _ = flags.GetString("answer-file")
	)

	output.InvocationInfo(ctx, cmd, args)

	if len(paths) == 1 && fileName == "" {
		fileName = filepath.Base(paths[0])
	}

	if fileName == "" {
		fileName = namesgenerator.GetRandomName(0)
	}

	webRTCSignallingDir, _ := flags.GetString("webrtc-signalling-dir")
	webRTCSignallingURL, _ := flags.GetString("webrtc-signalling-server-url")
	if err := c.configureWebRTC(flags); err != nil {
		return err
	}

	transport, err := client.NewTransport(c.webrtcConfig)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	var signaller sdp.ClientSignaller
	if webRTCSignallingDir != "" && webRTCSignallingURL != "" {
		return fmt.Errorf("cannot use both --webrtc-signalling-dir and --webrtc-signalling-server-url")
	}
	if webRTCSignallingDir == "" && webRTCSignallingURL == "" {
		return fmt.Errorf("must specify either --webrtc-signalling-dir or --webrtc-signalling-server-url")
	}

	if webRTCSignallingDir != "" {
		signaller = sdp.NewFileClientSignaller(offerFilePath, answerFilePath)
	} else {
		signaller = sdp.NewServerClientSignaller(webRTCSignallingURL, nil)
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

	fileReport.TransferEndTime = time.Now()
	if buf != nil {
		fileReport.TransferSize = int64(buf.Len())
		fileReport.Content = buf.Bytes()
	}
	events.Raise(ctx, &fileReport)

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("failed to send file: %s", resp.Status)
	} else {
		events.Success(ctx)
	}
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
	urls, _ := flags.GetStringSlice("webrtc-ice-servers")
	if len(urls) == 0 {
		urls = ice.STUNServerURLS
	}

	c.webrtcConfig = &webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: urls,
			},
		},
	}

	return nil
}
