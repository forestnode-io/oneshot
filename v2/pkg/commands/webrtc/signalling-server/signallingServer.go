package signallingserver

import (
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/signalling-server/start"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/signalling-server/stop"
	"github.com/spf13/cobra"
)

type Cmd struct {
	cobraCommand *cobra.Command
}

func New() *Cmd {
	return &Cmd{}
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "signalling-server",
		Short: "WebRTC signalling server",
		Long:  "WebRTC signalling server",
	}

	c.cobraCommand.AddCommand(subCommands()...)

	return c.cobraCommand
}

func subCommands() []*cobra.Command {
	return []*cobra.Command{
		start.New().Cobra(),
		stop.New().Cobra(),
	}
}
