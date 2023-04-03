package p2p

import (
	browserclient "github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p/browser-client"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p/client"
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
		Use:   "p2p",
		Short: "Peer-to-peer commands",
		Long:  "Peer-to-peer commands",
	}

	c.cobraCommand.AddCommand(subCommands()...)

	return c.cobraCommand
}

func subCommands() []*cobra.Command {
	return []*cobra.Command{
		client.New().Cobra(),
		browserclient.New().Cobra(),
	}
}
