package receive

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/file"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/client"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

func New() *Cmd {
	return &Cmd{}
}

type Cmd struct {
	cobraCommand       *cobra.Command
	fileTransferConfig *file.WriteTransferConfig
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
	c.cobraCommand.MarkFlagRequired("offer-file")
	flags.StringP("answer-file", "A", "", "Path to file which the SDP answer should be written to.")

	return c.cobraCommand
}

func (c *Cmd) receive(cmd *cobra.Command, args []string) error {
	var (
		ctx = cmd.Context()

		flags             = cmd.Flags()
		offerFilePath, _  = flags.GetString("offer-file")
		answerFilePath, _ = flags.GetString("answer-file")
	)

	output.InvocationInfo(ctx, cmd, args)

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

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	if err != nil {
		return err
	}
	req.Close = true

	httpClient := http.Client{
		Transport: &transport,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
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
		// TODO(raphaelreyna): get the host and size here
		"",
		int64(0),
	)
	defer cancelProgDisp()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to receive file: %s", resp.Status)
	}

	n, err := io.Copy(wts, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response body to file after %d bytes: %w", n, err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("failed to receive file: %s", resp.Status)
	} else {
		events.Success(ctx)
	}
	events.Stop(ctx)

	return err
}
