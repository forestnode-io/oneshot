package get

import (
	"os"

	"github.com/oneshot-uno/oneshot/v2/pkg/configuration"
	"github.com/oneshot-uno/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
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
		Use:   "get path",
		Short: "Get an individual value from a oneshot configuration file.",
		Long:  "Get an individual value from a oneshot configuration file.",
		RunE:  c.run,
		Args:  cobra.ExactArgs(1),
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)

	return c.cobraCommand
}

func (c *Cmd) run(cmd *cobra.Command, args []string) error {
	configData := viper.Get(args[0])
	if configData == nil {
		return output.UsageErrorF("no such key: %s", args[0])
	}

	return yaml.NewEncoder(os.Stdout).Encode(configData)
}
