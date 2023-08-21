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
	StatusCode     int                 `mapstructure:"status" yaml:"status"`
	Method         string              `mapstructure:"method" yaml:"method"`
	MatchHost      bool                `mapstructure:"matchhost" yaml:"matchhost"`
	Tee            bool                `mapstructure:"tee" yaml:"tee"`
	SpoofHost      string              `mapstructure:"spoofhost" yaml:"spoofhost"`
	RequestHeader  flagargs.HTTPHeader `mapstructure:"requestheader" yaml:"requestheader"`
	ResponseHeader flagargs.HTTPHeader `mapstructure:"responseheader" yaml:"responseheader"`
}

func (c *Configuration) Validate() error {
	if t := http.StatusText(c.StatusCode); t == "" && c.StatusCode != 0 {
		return fmt.Errorf("invalid status code")
	}

	return nil
}

func (c *Configuration) Hydrate() error {
	return nil
}

func SetFlags(cmd *cobra.Command) {
	fs := pflag.NewFlagSet("send flags", pflag.ContinueOnError)
	defer cmd.Flags().AddFlagSet(fs)

	flags.Int(fs, "cmd.rproxy.status", "status-code", "HTTP status code to send to client.")
	flags.String(fs, "cmd.rproxy.method", "method", "HTTP method to send to client.")
	flags.Bool(fs, "cmd.rproxy.matchhost", "match-host", `The 'Host' header will be set to match the host being reverse-proxied to.`)
	flags.Bool(fs, "cmd.rproxy.tee", "tee", `Send a copy of the proxied response to the console.`)
	flags.String(fs, "cmd.rproxy.spoofhost", "spoof-host", `Spoof the request host, the 'Host' header will be set to this value.
This Flag is ignored if the --match-host flag is set.`)
	flags.StringSlice(fs, "cmd.rproxy.requestheader", "request-header", `Header to send with the proxied request. Can be specified multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)
	flags.StringSlice(fs, "cmd.rproxy.responseheader", "response-header", `Header to send to send with the proxied response. Can be specified multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)

	cobra.AddTemplateFunc("sendFlags", func() *pflag.FlagSet {
		return fs
	})
}
