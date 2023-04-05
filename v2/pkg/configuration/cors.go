package configuration

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CORS struct {
	AllowedOrigins      []string `mapstructure:"allowedOrigins" yaml:"allowedOrigins"`
	AllowedHeaders      []string `mapstructure:"allowedHeaders" yaml:"allowedHeaders"`
	MaxAge              int      `mapstructure:"maxAge" yaml:"maxAge"`
	AllowCredentials    bool     `mapstructure:"allowCredentials" yaml:"allowCredentials"`
	AllowPrivateNetwork bool     `mapstructure:"allowPrivateNetwork" yaml:"allowPrivateNetwork"`
	SuccessStatus       int      `mapstructure:"successStatus" yaml:"successStatus"`

	fs *pflag.FlagSet
}

func (c *CORS) init() {
	c.fs = pflag.NewFlagSet("CORS Flags", pflag.ExitOnError)

	c.fs.StringSlice("cors-allowed-origins", []string{}, `Comma separated list of allowed origins.
	An allowed origin may be a domain name, or a wildcard (*).
	A domain name may contain a wildcard (*).`)
	c.fs.StringSlice("cors-allowed-headers", []string{}, `Comma separated list of allowed headers.
	An allowed header may be a header name, or a wildcard (*).
	If a wildcard (*) is used, all headers will be allowed.`)
	c.fs.Int("cors-max-age", 0, "How long the preflight results can be cached by the client.")
	c.fs.Bool("cors-allow-credentials", false, `Allow credentials like cookies, basic auth headers, and ssl certs for CORS requests.`)
	c.fs.Bool("cors-allow-private-network", false, `Allow private network requests from CORS requests.`)
	c.fs.Int("cors-success-status", 0, `HTTP status code to return for successful CORS requests.`)

	cobra.AddTemplateFunc("corsAllowedOrigins", func() []string {
		return c.AllowedOrigins
	})
}

func (c *CORS) setFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *CORS) mergeFlags() {
	if c.fs.Changed("cors-allowed-origins") {
		c.AllowedOrigins, _ = c.fs.GetStringSlice("cors-allowed-origins")
	}
	if c.fs.Changed("cors-allowed-headers") {
		c.AllowedHeaders, _ = c.fs.GetStringSlice("cors-allowed-headers")
	}
	if c.fs.Changed("cors-max-age") {
		c.MaxAge, _ = c.fs.GetInt("cors-max-age")
	}
	if c.fs.Changed("cors-allow-credentials") {
		c.AllowCredentials, _ = c.fs.GetBool("cors-allow-credentials")
	}
	if c.fs.Changed("cors-allow-private-network") {
		c.AllowPrivateNetwork, _ = c.fs.GetBool("cors-allow-private-network")
	}
	if c.fs.Changed("cors-success-status") {
		c.SuccessStatus, _ = c.fs.GetInt("cors-success-status")
	}
}

func (c *CORS) validate() error {
	return nil
}
