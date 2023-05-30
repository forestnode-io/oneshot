package config

import (
	"github.com/oneshot-uno/oneshot/v2/pkg/commands/config/get"
	"github.com/oneshot-uno/oneshot/v2/pkg/commands/config/path"
	"github.com/oneshot-uno/oneshot/v2/pkg/commands/config/set"
	"github.com/oneshot-uno/oneshot/v2/pkg/configuration"
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
		Use:     "config",
		Aliases: []string{"conf, configuration"},
		Short:   "Modify oneshot configuration files.",
		Long:    "Modify oneshot configuration files.",
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)

	c.cobraCommand.AddCommand(subCommands(c.config)...)

	return c.cobraCommand
}

func subCommands(config *configuration.Root) []*cobra.Command {
	return []*cobra.Command{
		get.New(config).Cobra(),
		set.New(config).Cobra(),
		path.New().Cobra(),
	}
}
