package configuration

import (
	"fmt"
	"net/http"
	"os"

	oneshothttp "github.com/oneshot-uno/oneshot/v2/pkg/net/http"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	CSRFToken    string              `mapstructure:"csrfToken" yaml:"csrfToken"`
	EOL          string              `mapstructure:"eol" yaml:"eol"`
	UI           string              `mapstructure:"ui" yaml:"ui"`
	DecodeBase64 bool                `mapstructure:"decodeBase64" yaml:"decodeBase64"`
	StatusCode   int                 `mapstructure:"statusCode" yaml:"statusCode"`
	Header       map[string][]string `mapstructure:"header" yaml:"header"`
	IncludeBody  bool                `mapstructure:"includeBody" yaml:"includeBody"`

	fs *pflag.FlagSet
}

func (c *Configuration) Init() {
	c.fs = pflag.NewFlagSet("receive flags", pflag.ExitOnError)

	c.fs.String("csrf-token", "", "Use a CSRF token, if left empty, a random token will be generated.")
	c.fs.String("eol", "unix", `How to parse EOLs in the received file.
Acceptable values are 'unix' and 'dos'; 'unix': '\n', 'dos': '\r\n'.`)
	c.fs.StringP("ui", "U", "", "Name of ui file to use.")
	c.fs.Bool("decode-b64", false, "Decode base-64.")
	c.fs.Int("status-code", 200, "HTTP status code sent to client.")
	c.fs.StringSliceP("header", "H", nil, `Header to send to client. Can be specified multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)
	c.fs.Bool("include-body", false, "Include the request body in the report. If not using json output, this will be ignored.")

	cobra.AddTemplateFunc("receiveFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *Configuration) SetFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *Configuration) MergeFlags() {
	if c.fs.Changed("csrf-token") {
		c.CSRFToken, _ = c.fs.GetString("csrf-token")
	}
	if c.fs.Changed("eol") {
		c.EOL, _ = c.fs.GetString("eol")
	}
	if c.fs.Changed("ui") {
		c.UI, _ = c.fs.GetString("ui")
	}
	if c.fs.Changed("decode-b64") {
		c.DecodeBase64, _ = c.fs.GetBool("decode-b64")
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
	if c.fs.Changed("include-body") {
		c.IncludeBody, _ = c.fs.GetBool("include-body")
	}
}

func (c *Configuration) Validate() error {
	if (c.EOL != "unix" && c.EOL != "dos") && c.EOL != "" {
		return fmt.Errorf("invalid eol: %s", c.EOL)
	}

	if c.UI != "" {
		stat, err := os.Stat(c.UI)
		if err != nil {
			return fmt.Errorf("invalid ui file: %w", err)
		}
		if stat.IsDir() {
			return fmt.Errorf("invalid ui file: %s is a directory", c.UI)
		}
	}

	if t := http.StatusText(c.StatusCode); t == "" {
		return fmt.Errorf("invalid status code")
	}

	return nil
}

func (c *Configuration) Hydrate() error {
	return nil
}
