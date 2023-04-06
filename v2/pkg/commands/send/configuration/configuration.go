package configuration

import (
	"net/http"
	"runtime"

	"github.com/raphaelreyna/oneshot/v2/pkg/flagargs"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	ArchiveMethod flagargs.ArchiveMethod `mapstructure:"archiveMethod" yaml:"archiveMethod"`
	NoDownload    bool                   `mapstructure:"noDownload" yaml:"noDownload"`
	MIME          string                 `mapstructure:"mime" yaml:"mime"`
	Name          string                 `mapstructure:"name" yaml:"name"`
	StatusCode    int                    `mapstructure:"statusCode" yaml:"statusCode"`
	Header        map[string][]string    `mapstructure:"header" yaml:"header"`

	fs *pflag.FlagSet
}

func (c *Configuration) Init() {
	c.fs = pflag.NewFlagSet("send flags", pflag.ExitOnError)

	var amf flagargs.ArchiveMethod
	c.fs.VarP(&amf, "archive-method", "a", `Which archive method to use when sending directories.
Recognized values are 'zip', 'tar' and 'tar.gz'.`)
	if runtime.GOOS == "windows" {
		c.fs.Lookup("archive-method").DefValue = "zip"
	} else {
		c.fs.Lookup("archive-method").DefValue = "tar.gz"
	}
	c.fs.BoolP("no-download", "D", false, "Do not allow the client to download the file.")
	c.fs.StringP("mime", "m", "", `MIME type of file presented to client.
If not set, either no MIME type or the mime/type of the file will be user, depending on of a file was given.`)
	c.fs.StringP("name", "n", "", `Name of file presented to client if downloading.
If not set, either a random name or the name of the file will be used, depending on if a file was given.`)
	c.fs.Int("status-code", http.StatusOK, "HTTP status code to send to client.")
	c.fs.StringSliceP("header", "H", nil, `Header to send to client. Can be specified multiple times. 
Format: <HEADER NAME>=<HEADER VALUE>`)

	cobra.AddTemplateFunc("sendFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *Configuration) SetFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *Configuration) MergeFlags() {
	if c.fs.Changed("archive-method") {
		am, ok := c.fs.Lookup("archive-method").Value.(*flagargs.ArchiveMethod)
		if !ok {
			panic("archive-method flag is not an archiveFlag")
		}
		c.ArchiveMethod = *am
	}
	if c.fs.Changed("no-download") {
		c.NoDownload, _ = c.fs.GetBool("no-download")
	}
	if c.fs.Changed("mime") {
		c.MIME, _ = c.fs.GetString("mime")
	}
	if c.fs.Changed("name") {
		c.Name, _ = c.fs.GetString("name")
	}
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
