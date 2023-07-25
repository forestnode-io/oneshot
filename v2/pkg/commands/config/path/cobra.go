package path

import (
	"fmt"

	"github.com/forestnode-io/oneshot/v2/pkg/configuration"
	"github.com/forestnode-io/oneshot/v2/pkg/output"
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
		Use:     "path",
		Aliases: []string{"location", "file"},
		Short:   "Get the path to the oneshot configuration file being used.",
		Long:    "Get the path to the oneshot configuration file being used.",
		RunE:    c.run,
		Args:    cobra.NoArgs,
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)

	return c.cobraCommand
}

func (c *Cmd) run(cmd *cobra.Command, args []string) error {
	if configuration.ConfigPath() != "" {
		fmt.Printf("%s\n", configuration.ConfigPath())
	} else {
		return output.UsageErrorF("no configuration file found")
	}
	return nil
}
