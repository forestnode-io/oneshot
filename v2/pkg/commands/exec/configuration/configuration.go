package configuration

import (
	"fmt"
	"os"

	oneshothttp "github.com/oneshot-uno/oneshot/v2/pkg/net/http"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	EnforceCGI     bool                `mapstructure:"enforceCGI" yaml:"enforceCGI"`
	Env            []string            `mapstructure:"env" yaml:"env"`
	Dir            string              `mapstructure:"dir" yaml:"dir"`
	StdErr         string              `mapstructure:"stderr" yaml:"stderr"`
	ReplaceHeaders bool                `mapstructure:"replaceHeaders" yaml:"replaceHeaders"`
	Header         map[string][]string `mapstructure:"headers" yaml:"headers"`

	fs *pflag.FlagSet
}

func (c *Configuration) Init() {
	c.fs = pflag.NewFlagSet("exec flags", pflag.ContinueOnError)

	c.fs.BoolVar(&c.EnforceCGI, "enforce-cgi", false, "The exec must conform to the CGI standard.")
	c.fs.StringSliceP("env", "e", []string{}, "Set an environment variable.")
	c.fs.String("dir", "", "Set the working directory.")
	c.fs.String("stderr", "", "Where to send exec stderr.")
	c.fs.Bool("replace-headers", false, "Allow command to replace header values.")
	c.fs.StringSliceP("header", "H", nil, `Header to send to client. Can be specified multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)

	cobra.AddTemplateFunc("execFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *Configuration) SetFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *Configuration) MergeFlags() {
	if c.fs.Changed("enforce-cgi") {
		c.EnforceCGI, _ = c.fs.GetBool("enforce-cgi")
	}
	if c.fs.Changed("env") {
		c.Env, _ = c.fs.GetStringSlice("env")
	}
	if c.fs.Changed("dir") {
		c.Dir, _ = c.fs.GetString("dir")
	}
	if c.fs.Changed("stderr") {
		c.StdErr, _ = c.fs.GetString("stderr")
	}
	if c.fs.Changed("replace-headers") {
		c.ReplaceHeaders, _ = c.fs.GetBool("replace-headers")
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
	if c.Dir != "" {
		stat, err := os.Stat(c.Dir)
		if err != nil {
			return fmt.Errorf("invalid directory: %w", err)
		}
		if !stat.IsDir() {
			return fmt.Errorf("invalid directory: %s is not a directory", c.Dir)
		}
	}
	return nil
}

func (c *Configuration) Hydrate() error {
	return nil
}
