package configuration

import (
	"fmt"
	"net/http"

	"github.com/oneshot-uno/oneshot/v2/pkg/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	ArchiveMethod string              `mapstructure:"archivemethod" yaml:"archivemethod"`
	NoDownload    bool                `mapstructure:"nodownload" yaml:"nodownload"`
	MIME          string              `mapstructure:"mime" yaml:"mime"`
	Name          string              `mapstructure:"name" yaml:"name"`
	StatusCode    int                 `mapstructure:"status" yaml:"status"`
	Header        map[string][]string `mapstructure:"header" yaml:"header"`
}

func SetFlags(cmd *cobra.Command) {
	fs := pflag.NewFlagSet("send flags", pflag.ExitOnError)
	defer cmd.Flags().AddFlagSet(fs)

	flags.StringP(fs, "cmd.send.archivemethod", "archive-method", "a", `Which archive method to use when sending directories.`)
	flags.BoolP(fs, "cmd.send.nodownload", "no-download", "D", "Do not allow the client to download the file.")
	flags.StringP(fs, "cmd.send.mime", "mime", "m", `MIME type of file presented to client.`)
	flags.StringP(fs, "cmd.send.name", "name", "n", `Name of file presented to client if downloading.`)
	flags.Int(fs, "cmd.send.status", "status-code", "HTTP status code to send to client.")
	flags.StringSliceP(fs, "cmd.send.header", "header", "H", `Header to send to client. Can be specified multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)

	cobra.AddTemplateFunc("sendFlags", func() *pflag.FlagSet {
		return fs
	})
}

func (c *Configuration) Validate() error {
	if t := http.StatusText(c.StatusCode); t == "" {
		return fmt.Errorf("invalid status code")
	}
	return nil
}

func (c *Configuration) Hydrate() error {
	return nil
}
