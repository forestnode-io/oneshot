package client

import (
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/client/receive"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/client/send"
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
		Use:   "client",
		Short: "WebRTC client commands",
		Long:  "WebRTC client commands",
	}

	c.cobraCommand.AddCommand(subCommands()...)

	return c.cobraCommand
}

func subCommands() []*cobra.Command {
	return []*cobra.Command{
		send.New().Cobra(),
		receive.New().Cobra(),
	}
}
