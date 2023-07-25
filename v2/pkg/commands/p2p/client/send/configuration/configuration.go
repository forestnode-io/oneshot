package configuration

import (
	"github.com/forestnode-io/oneshot/v2/pkg/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	Name          string `json:"name" yaml:"name"`
	ArchiveMethod string `json:"archivemethod" yaml:"archivemethod"`
}

func (c *Configuration) Validate() error {
	return nil
}

func SetFlags(cmd *cobra.Command) {
	fs := pflag.NewFlagSet("send flags", pflag.ExitOnError)
	defer cmd.Flags().AddFlagSet(fs)

	flags.StringP(fs, "cmd.p2p.client.send.name", "name", "n", "Name of file presented to the server.")
	flags.StringP(fs, "cmd.p2p.client.send.archivemethod", "archive-method", "a", `Which archive method to use when sending directories.
Recognized values are "zip", "tar" and "tar.gz".`)

	cobra.AddTemplateFunc("sendFlags", func() *pflag.FlagSet {
		return fs
	})
}
