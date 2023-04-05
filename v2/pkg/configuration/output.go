package configuration

import (
	"github.com/raphaelreyna/oneshot/v2/pkg/flagargs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Output struct {
	Quiet   bool                  `mapstructure:"quiet" yaml:"quiet"`
	Format  flagargs.OutputFormat `mapstructure:"format" yaml:"format"`
	QRCode  bool                  `mapstructure:"qrCode" yaml:"qrCode"`
	NoColor bool                  `mapstructure:"noColor" yaml:"noColor"`

	fs *pflag.FlagSet
}

func (c *Output) init() {
	c.fs = pflag.NewFlagSet("Output Flags", pflag.ExitOnError)

	c.fs.BoolP("quiet", "q", false, "Disable all output except for received data")
	var format flagargs.OutputFormat
	c.fs.VarP(&format, "output", "o", `Set output format. Valid formats are: json[=opts].
	Valid json opts are:
		- compact
			Disables tabbed, pretty printed json.
		- include-file-contents
			Includes the contents of files in the json output.
			This is on by default when sending from stdin or receiving to stdout.
		- exclude-file-contents
			Excludes the contents of files in the json output.
			This is on by default when sending or receiving to or from disk.`)
	c.fs.Bool("qr-code", false, "Print a QR code of a URL that the server can be reached at")
	c.fs.Bool("no-color", false, "Disable color output")

	cobra.AddTemplateFunc("outputFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *Output) setFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *Output) mergeFlags() {
	if c.fs.Changed("quiet") {
		c.Quiet, _ = c.fs.GetBool("quiet")
	}
	if c.fs.Changed("output") {
		offa, ok := c.fs.Lookup("output").Value.(*flagargs.OutputFormat)
		if !ok {
			panic("invalid type for output flag")
		}
		c.Format = *offa
	}
	if c.fs.Changed("qr-code") {
		c.QRCode, _ = c.fs.GetBool("qr-code")
	}
	if c.fs.Changed("no-color") {
		c.NoColor, _ = c.fs.GetBool("no-color")
	}
}

func (c *Output) validate() error {
	return nil
}
