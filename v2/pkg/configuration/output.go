package configuration

import (
	"fmt"
	"strings"

	"github.com/oneshot-uno/oneshot/v2/pkg/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Output struct {
	Quiet   bool   `mapstructure:"quiet" yaml:"quiet"`
	Format  string `mapstructure:"format" yaml:"format"`
	QRCode  bool   `mapstructure:"qrCode" yaml:"qrCode"`
	NoColor bool   `mapstructure:"noColor" yaml:"noColor"`
}

func setOutputFlags(cmd *cobra.Command) {
	fs := pflag.NewFlagSet("Output Flags", pflag.ExitOnError)
	defer cmd.PersistentFlags().AddFlagSet(fs)

	flags.BoolP(fs, "output.quiet", "quiet", "q", "Disable all output except for received data")
	flags.StringP(fs, "output.format", "o", "", `Set output format. Valid formats are: json[=opts].
Valid json opts are:
	- compact
		Disables tabbed, pretty printed json.
	- include-file-contents
		Includes the contents of files in the json output.
		This is on by default when sending from stdin or receiving to stdout.
	- exclude-file-contents
		Excludes the contents of files in the json output.
		This is on by default when sending or receiving to or from disk.`)
	flags.Bool(fs, "output.qrCode", "qr-code", "Print a QR code of a URL that the server can be reached at")
	flags.Bool(fs, "output.noColor", "no-color", "Disable color output")

	cobra.AddTemplateFunc("outputFlags", func() *pflag.FlagSet {
		return fs
	})
	cobra.AddTemplateFunc("outputClientFlags", func() *pflag.FlagSet {
		fs := pflag.NewFlagSet("Output Flags", pflag.ExitOnError)
		fs.BoolP("quiet", "q", false, "Disable all output except for received data")
		fs.StringP("output", "o", "", `Set output format. Valid formats are: json[=opts].
Valid json opts are:
	- compact
		Disables tabbed, pretty printed json.
	- include-file-contents
		Includes the contents of files in the json output.
		This is on by default when sending from stdin or receiving to stdout.
	- exclude-file-contents
		Excludes the contents of files in the json output.
		This is on by default when sending or receiving to or from disk.`)
		fs.Bool("no-color", false, "Disable color output")
		return fs
	})
}

func (c *Output) validate() error {
	if c.Format == "" {
		return nil
	}
	parts := strings.Split(c.Format, "=")
	if len(parts) == 0 || 2 < len(parts) {
		return fmt.Errorf("invalid output format: %s", c.Format)
	}
	format := parts[0]
	if format != "json" {
		return fmt.Errorf("invalid output format: %s", c.Format)
	}

	opts := []string{}
	if len(parts) == 2 {
		opts = strings.Split(parts[1], ",")
	}

	for _, opt := range opts {
		if opt != "compact" && opt != "include-file-contents" && opt != "exclude-file-contents" {
			return fmt.Errorf("invalid output format option: %s", opt)
		}
	}

	return nil
}
