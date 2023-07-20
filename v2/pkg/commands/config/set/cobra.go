package set

import (
	"strconv"
	"strings"

	"github.com/oneshot-uno/oneshot/v2/pkg/configuration"
	"github.com/oneshot-uno/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		Use:   "set path value...",
		Short: "Set an individual value in a oneshot configuration file.",
		Long:  "Set an individual value in a oneshot configuration file.",
		RunE:  c.run,
		Args:  cobra.MinimumNArgs(2),
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)

	return c.cobraCommand
}

func (c *Cmd) run(cmd *cobra.Command, args []string) error {
	if configuration.ConfigPath() == "" {
		return output.UsageErrorF("no configuration file found")
	}

	v := viper.Get(args[0])
	if v == nil {
		return output.UsageErrorF("no such key: %s", args[0])
	}
	switch v.(type) {
	case string:
		viper.Set(args[0], args[1])
	case int:
		x, err := strconv.Atoi(args[1])
		if err != nil {
			return output.UsageErrorF("failed to convert value to int: %w", err)
		}
		viper.Set(args[0], x)
	case bool:
		x, err := strconv.ParseBool(args[1])
		if err != nil {
			return output.UsageErrorF("failed to convert value to bool: %w", err)
		}
		viper.Set(args[0], x)
	case []string:
		if 3 <= len(args) {
			viper.Set(args[0], args[1:])
		} else {
			viper.Set(args[0], strings.Split(args[1], ","))
		}
	case []int:
		if 3 <= len(args) {
			ints := make([]int, len(args)-1)
			var err error
			for i := range args[1:] {
				ints[i], err = strconv.Atoi(args[i+1])
				if err != nil {
					return output.UsageErrorF("failed to convert value to int: %w", err)
				}
			}
			viper.Set(args[0], ints)
		}
	default:
		return output.UsageErrorF("unsupported type: %T", v)
	}

	err := viper.WriteConfig()
	if err != nil {
		return output.UsageErrorF("failed to write configuration file: %w", err)
	}

	return nil
}
