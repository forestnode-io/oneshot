package p2p

import (
	browserclient "github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p/browser-client"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p/client"
	"github.com/spf13/cobra"
)

type Configuration struct {
	BrowsertClient browserclient.Configuration
}

func New(config *Configuration) *Cmd {
	return &Cmd{}
}

type Cmd struct {
	cobraCommand *cobra.Command
	config       *Configuration
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

func subCommands(config *Configuration) []*cobra.Command {
	return []*cobra.Command{
		client.New().Cobra(),
		browserclient.New(&config.BrowsertClient).Cobra(),
	}
}
