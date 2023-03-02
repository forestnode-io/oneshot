package send

import (
	"fmt"
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
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

func New() *Cmd {
	return &Cmd{}
}

type Cmd struct {
	cobraCommand  *cobra.Command
	archiveMethod archiveFlag
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
	c.cobraCommand.MarkFlagRequired("offer-file")
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

	transport := client.Transport{
		Config: &webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
			},
		},
	}
	signaller := sdp.NewFileClientSignaller(offerFilePath, answerFilePath)
	signaller.Start(ctx, &transport)
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

	<-transport.ConnectionEstablished()

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
		Transport: &transport,
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
