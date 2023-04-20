package configuration

import (
	"runtime"

	"github.com/oneshot-uno/oneshot/v2/pkg/flagargs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	Name          string                 `json:"name" yaml:"name"`
	ArchiveMethod flagargs.ArchiveMethod `json:"archiveMethod" yaml:"archiveMethod"`

	fs *pflag.FlagSet
}

func (c *Configuration) Init() {
	c.fs = pflag.NewFlagSet("send flags", pflag.ExitOnError)

	c.fs.StringP("name", "n", "", "Name of file presented to the server.")
	var archiveMethod flagargs.ArchiveMethod
	c.fs.VarP(&archiveMethod, "archive-method", "a", `Which archive method to use when sending directories.
Recognized values are "zip", "tar" and "tar.gz".`)
	if runtime.GOOS == "windows" {
		c.fs.Lookup("archive-method").DefValue = "zip"
	} else {
		c.fs.Lookup("archive-method").DefValue = "tar.gz"
	}

	cobra.AddTemplateFunc("sendFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *Configuration) SetFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *Configuration) MergeFlags() {
	if c.fs.Changed("name") {
		c.Name, _ = c.fs.GetString("name")
	}
	if c.fs.Changed("archive-method") {
		am, ok := c.fs.Lookup("archive-method").Value.(*flagargs.ArchiveMethod)
		if !ok {
			panic("archive-method flag is not an archiveFlag")
		}
		c.ArchiveMethod = *am
	}
}

func (c *Configuration) Validate() error {
	return nil
}
