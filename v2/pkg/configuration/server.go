package configuration

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Server struct {
	Host        string        `mapstructure:"host" yaml:"host"`
	Port        int           `mapstructure:"port" yaml:"port"`
	Timeout     time.Duration `mapstructure:"timeout" yaml:"timeout"`
	AllowBots   bool          `mapstructure:"allowBots" yaml:"allowBots"`
	MaxReadSize SizeFlagArg   `mapstructure:"maxReadSize" yaml:"maxReadSize"`
	ExitOnFail  bool          `mapstructure:"exitOnFail" yaml:"exitOnFail"`
	TLSCert     string        `mapstructure:"tlsCert" yaml:"tlsCert"`
	TLSKey      string        `mapstructure:"tlsKey" yaml:"tlsKey"`

	fs *pflag.FlagSet
}

func (c *Server) init() {
	c.fs = pflag.NewFlagSet("Server Flags", pflag.ExitOnError)

	c.fs.String("host", "", "Host to listen on")
	c.fs.IntP("port", "p", 8080, "Port to listen on")
	c.fs.Duration("timeout", 0, `How long to wait for a connection to be established before timing out.
A value of 0 will cause oneshot to wait indefinitely.`)
	c.fs.Bool("allow-bots", false, "Allow bots access")
	var maxReadSize SizeFlagArg
	c.fs.Var(&maxReadSize, "max-read-size", `Maximum read size for incoming request bodies. A value of zero will cause oneshot to read until EOF.
	Format is a number followed by a unit of measurement.
	Valid units are: b, B,
		Kb, KB, KiB,
		Mb, MB, MiB,
		Gb, GB, GiB,
		Tb, TB, TiB
	Example: 1.5GB`)
	c.fs.Bool("exit-on-fail", false, "Exit after a failed transfer, without waiting for a new connection")
	c.fs.String("tls-cert", "", "Path to TLS certificate")
	c.fs.String("tls-key", "", "Path to TLS key")

	cobra.AddTemplateFunc("serverFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *Server) setFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *Server) mergeFlags() {
	if c.fs.Changed("host") {
		c.Host, _ = c.fs.GetString("host")
	}
	if c.fs.Changed("port") {
		c.Port, _ = c.fs.GetInt("port")
	}
	if c.fs.Changed("timeout") {
		c.Timeout, _ = c.fs.GetDuration("timeout")
	}
	if c.fs.Changed("allow-bots") {
		c.AllowBots, _ = c.fs.GetBool("allow-bots")
	}
	if c.fs.Changed("max-read-size") {
		mrs, ok := c.fs.Lookup("max-read-size").Value.(*SizeFlagArg)
		if !ok {
			panic("max-read-size flag is not a SizeFlagArg")
		}
		c.MaxReadSize = *mrs
	}
	if c.fs.Changed("exit-on-fail") {
		c.ExitOnFail, _ = c.fs.GetBool("exit-on-fail")
	}
	if c.fs.Changed("tls-cert") {
		c.TLSCert, _ = c.fs.GetString("tls-cert")
	}
	if c.fs.Changed("tls-key") {
		c.TLSKey, _ = c.fs.GetString("tls-key")
	}
}

func (c *Server) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.TLSCert != "" && c.TLSKey == "" {
		return fmt.Errorf("tls-key is required when tls-cert is set")
	}
	if c.TLSKey != "" && c.TLSCert == "" {
		return fmt.Errorf("tls-cert is required when tls-key is set")
	}

	return nil
}
