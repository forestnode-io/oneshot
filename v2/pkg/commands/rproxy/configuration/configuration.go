package configuration

import (
	"fmt"
	"net/http"

	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	StatusCode     int                 `mapstructure:"statusCode" yaml:"statusCode"`
	Method         string              `mapstructure:"method" yaml:"method"`
	MatchHost      bool                `mapstructure:"matchHost" yaml:"matchHost"`
	Tee            bool                `mapstructure:"tee" yaml:"tee"`
	SpoofHost      string              `mapstructure:"spoofHost" yaml:"spoofHost"`
	RequestHeader  map[string][]string `mapstructure:"requestHeader" yaml:"requestHeader"`
	ResponseHeader map[string][]string `mapstructure:"responseHeader" yaml:"responseHeader"`

	fs *pflag.FlagSet
}

func (c *Configuration) Init() {
	c.fs = pflag.NewFlagSet("send flags", pflag.ExitOnError)

	c.fs.Int("status-code", 0, "HTTP status code to send with the proxied response.")
	c.fs.String("method", "", "HTTP method to send with the proxied request.")
	c.fs.Bool("match-host", false, `The 'Host' header will be set to match the host being reverse-proxied to.`)
	c.fs.Bool("tee", false, `Send a copy of the proxied response to the console.`)
	c.fs.String("spoof-host", "", `Spoof the request host, the 'Host' header will be set to this value.
This Flag is ignored if the --match-host flag is set.`)
	c.fs.StringSlice("request-header", nil, `Header to send with the proxied request. Can be specified multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)
	c.fs.StringSlice("response-header", nil, `Header to send to send with the proxied response. Can be specified multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)

	cobra.AddTemplateFunc("sendFlags", func() *pflag.FlagSet {
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
	if c.fs.Changed("method") {
		c.Method, _ = c.fs.GetString("method")
	}
	if c.fs.Changed("match-host") {
		c.MatchHost, _ = c.fs.GetBool("match-host")
	}
	if c.fs.Changed("tee") {
		c.Tee, _ = c.fs.GetBool("tee")
	}
	if c.fs.Changed("spoof-host") {
		c.SpoofHost, _ = c.fs.GetString("spoof-host")
	}
	if c.fs.Changed("request-header") {
		header, _ := c.fs.GetStringSlice("request-header")
		hdr, err := oneshothttp.HeaderFromStringSlice(header)
		if err != nil {
			panic(err)
		}
		c.RequestHeader = hdr
	}
	if c.fs.Changed("response-header") {
		header, _ := c.fs.GetStringSlice("response-header")
		hdr, err := oneshothttp.HeaderFromStringSlice(header)
		if err != nil {
			panic(err)
		}
		c.ResponseHeader = hdr
	}
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
