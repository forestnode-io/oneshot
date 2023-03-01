package webrtc

import (
	browserclient "github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/browser-client"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/client"
	signallingserver "github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/signalling-server"
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
		Use:   "webrtc",
		Short: "WebRTC commands",
		Long:  "WebRTC commands",
	}

	c.cobraCommand.AddCommand(subCommands()...)

	return c.cobraCommand
}

func subCommands() []*cobra.Command {
	return []*cobra.Command{
		client.New().Cobra(),
		signallingserver.New().Cobra(),
		browserclient.New().Cobra(),
	}
}
