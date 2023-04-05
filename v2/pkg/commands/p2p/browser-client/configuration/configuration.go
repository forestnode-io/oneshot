package configuration

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	Open bool `json:"open" yaml:"open"`

	fs *pflag.FlagSet
}

func (c *Configuration) Init() {
	c.fs = pflag.NewFlagSet("browser client flags", pflag.ExitOnError)

	c.fs.Bool("open", false, "Open the browser client in a new tab.")

	cobra.AddTemplateFunc("browserClientFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *Configuration) SetFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *Configuration) MergeFlags() {
	if c.fs.Changed("open") {
		c.Open, _ = c.fs.GetBool("open")
	}
}

func (c *Configuration) Validate() error {
	return nil
}
