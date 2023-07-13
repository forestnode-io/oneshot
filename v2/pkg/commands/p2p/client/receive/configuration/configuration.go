package configuration

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct{}

func (c *Configuration) Validate() error {
	return nil
}

func SetFlags(cmd *cobra.Command) {
	fs := pflag.NewFlagSet("redirect flags", pflag.ContinueOnError)
	defer cmd.Flags().AddFlagSet(fs)

	cobra.AddTemplateFunc("redirectFlags", func() *pflag.FlagSet {
		return fs
	})
}
