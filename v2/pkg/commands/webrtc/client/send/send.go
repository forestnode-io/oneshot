package send

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/client"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/sdp"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

func New() *Cmd {
	return &Cmd{}
}

type Cmd struct {
	cobraCommand *cobra.Command
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

	var (
		body   io.Reader
		header = http.Header{}
	)
	// TODO(raphaelreyna): handle multiple files and dirs
	if len(paths) == 1 {
		header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		file, err := os.OpenFile(paths[0], os.O_RDONLY, 0)
		if err != nil {
			return err
		}
		defer file.Close()
		body = file
	} else {
		header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		body = os.Stdin
	}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080", body)
	if err != nil {
		return err
	}
	req.Header = header
	req.Close = true

	httpClient := http.Client{
		Transport: &transport,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("failed to send file: %s", resp.Status)
	} else {
		events.Success(ctx)
	}
	events.Stop(ctx)

	return err
}
