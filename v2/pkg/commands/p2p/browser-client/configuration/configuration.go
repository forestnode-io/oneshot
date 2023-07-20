package configuration

import (
	"github.com/oneshot-uno/oneshot/v2/pkg/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	Open bool `json:"open" yaml:"open"`
}

func (c *Configuration) Validate() error {
	return nil
}

func SetFlags(cmd *cobra.Command) {
	fs := pflag.NewFlagSet("p2p flags", pflag.ExitOnError)
	defer cmd.Flags().AddFlagSet(fs)

	flags.BoolP(fs, "cmd.p2p.browserclient.open", "open", "o", "Open the browser to the generated URL.")

	cobra.AddTemplateFunc("browserClientFlags", func() *pflag.FlagSet {
		return fs
	})
}
