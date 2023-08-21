package configuration

import (
	"fmt"
	"net/http"

	"github.com/forestnode-io/oneshot/v2/pkg/flagargs"
	"github.com/forestnode-io/oneshot/v2/pkg/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	StatusCode int                 `mapstructure:"status" yaml:"status"`
	Header     flagargs.HTTPHeader `mapstructure:"header" yaml:"header"`
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

func SetFlags(cmd *cobra.Command) {
	fs := pflag.NewFlagSet("redirect flags", pflag.ContinueOnError)
	defer cmd.Flags().AddFlagSet(fs)

	flags.Int(fs, "cmd.redirect.status", "status-code", "HTTP status code to send to client.")
	flags.StringSliceP(fs, "cmd.redirect.header", "header", "H", `Header to send to client. Can be specified multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)

	cobra.AddTemplateFunc("redirectFlags", func() *pflag.FlagSet {
		return fs
	})
}
