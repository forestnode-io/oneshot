package p2p

import (
	browserclient "github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p/browser-client"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p/client"
	"github.com/raphaelreyna/oneshot/v2/pkg/configuration"
	"github.com/spf13/cobra"
)

func New(config *configuration.Root) *Cmd {
	return &Cmd{
		config: config,
	}
}

type Cmd struct {
	cobraCommand *cobra.Command
	config       *configuration.Root
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

	c.cobraCommand.AddCommand(subCommands(c.config)...)

	return c.cobraCommand
}

func subCommands(config *configuration.Root) []*cobra.Command {
	return []*cobra.Command{
		client.New(config).Cobra(),
		browserclient.New(config).Cobra(),
	}
}
