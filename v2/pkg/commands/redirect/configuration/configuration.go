package configuration

import (
	"net/http"

	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	StatusCode int                 `mapstructure:"statusCode" yaml:"statusCode"`
	Header     map[string][]string `mapstructure:"header" yaml:"header"`

	fs *pflag.FlagSet
}

func (c *Configuration) Init() {
	c.fs = pflag.NewFlagSet("send flags", pflag.ExitOnError)

	c.fs.Int("status-code", http.StatusTemporaryRedirect, "HTTP status code to send to client.")
	c.fs.StringSliceP("header", "H", nil, `Header to send to client. Can be specified multiple times. 
Format: <HEADER NAME>=<HEADER VALUE>`)

	cobra.AddTemplateFunc("redirectFlags", func() *pflag.FlagSet {
		return c.fs
	})
}
func (c *Configuration) SetFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *Configuration) MergeFlags() {
	if c.fs.Changed("status-code") {
		c.StatusCode, _ = c.fs.GetInt("status-code")
	}
	if c.fs.Changed("header") {
		header, _ := c.fs.GetStringSlice("header")
		hdr, err := oneshothttp.HeaderFromStringSlice(header)
		if err != nil {
			panic(err)
		}
		c.Header = hdr
	}
}

func (c *Configuration) Validate() error {
	return nil
}

func (c *Configuration) Hydrate() error {
	return nil
}
