package configuration

import (
	"fmt"
	"os"

	"github.com/forestnode-io/oneshot/v2/pkg/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	EnforceCGI     bool                `mapstructure:"enforcecgi" yaml:"enforcecgi"`
	Env            []string            `mapstructure:"env" yaml:"env"`
	Dir            string              `mapstructure:"dir" yaml:"dir"`
	StdErr         string              `mapstructure:"stderr" yaml:"stderr"`
	ReplaceHeaders bool                `mapstructure:"replaceheaders" yaml:"replaceheaders"`
	Header         map[string][]string `mapstructure:"headers" yaml:"headers"`
}

func SetFlags(cmd *cobra.Command) {
	fs := pflag.NewFlagSet("exec flags", pflag.ContinueOnError)
	defer cmd.Flags().AddFlagSet(fs)

	flags.Bool(fs, "cmd.exec.enforcecgi", "enforce-cgi", "The exec must conform to the CGI standard.")
	flags.StringSliceP(fs, "cmd.exec.env", "env", "e", "Set an environment variable.")
	flags.String(fs, "cmd.exec.dir", "dir", "Set the working directory.")
	flags.String(fs, "cmd.exec.stderr", "stderr", "Where to send exec stderr.")
	flags.Bool(fs, "cmd.exec.replaceheaders", "replace-headers", "Allow command to replace header values.")
	flags.StringSliceP(fs, "cmd.exec.header", "header", "H", `Header to send to client. Can be specified multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)

	cobra.AddTemplateFunc("execFlags", func() *pflag.FlagSet {
		return fs
	})
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
