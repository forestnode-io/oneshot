package configuration

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	fs *pflag.FlagSet
}

func (c *Configuration) Init() {
	c.fs = pflag.NewFlagSet("receive flags", pflag.ExitOnError)

	cobra.AddTemplateFunc("receiveFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *Configuration) SetFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *Configuration) MergeFlags() {
}

func (c *Configuration) Validate() error {
	return nil
}
