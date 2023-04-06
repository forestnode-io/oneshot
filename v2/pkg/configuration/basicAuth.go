package configuration

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type BasicAuth struct {
	Username           string `mapstructure:"username" yaml:"username"`
	Password           string `mapstructure:"password" yaml:"password"`
	PasswordFile       string `mapstructure:"passwordFile" yaml:"passwordFile"`
	PasswordPrompt     bool   `mapstructure:"passwordPrompt" yaml:"passwordPrompt"`
	UnauthorizedPage   string `mapstructure:"unauthorizedPage" yaml:"unauthorizedPage"`
	UnauthorizedStatus int    `mapstructure:"unauthorizedStatus" yaml:"unauthorizedStatus"`
	NoDialog           bool   `mapstructure:"noDialog" yaml:"noDialog"`

	fs *pflag.FlagSet
}

func (c *BasicAuth) init() {
	c.fs = pflag.NewFlagSet("Basic Auth Flags", pflag.ExitOnError)

	c.fs.StringP("username", "u", "", `Username for basic authentication.
If a password is not also provided then the client may enter any password.`)
	c.fs.StringP("password", "P", "", `Password for basic authentication.
If a username is not also provided using the --username flag then the client may enter any username.
If either the --password-prompt or --password-file flags are set, this flag will be ignored.`)
	c.fs.String("password-file", "", `Path to file containing password for basic authentication.
If a username is not also provided then the client may enter any username.
If the --password-prompt flag is set, this flags will be ignored.`)
	c.fs.BoolP("password-prompt", "W", false, `Prompt for password for basic authentication.
If a username is not also provided then the client may enter any username.
If the --password-file flag is set, this flag will be ignored.`)
	c.fs.String("unauthorized-page", "", `Path to file containing HTML to display when a user is unauthorized.
If this flag is not set then a default page will be displayed.`)
	c.fs.Int("unauthorized-status", 401, `HTTP status code to return when a user is unauthorized.`)
	c.fs.Bool("no-dialog", false, `Do not display a dialog box when prompting for credentials.`)

	cobra.AddTemplateFunc("basicAuthFlags", func() *pflag.FlagSet {
		return c.fs
	})
	cobra.AddTemplateFunc("basicAuthClientFlags", func() *pflag.FlagSet {
		fs := pflag.NewFlagSet("Basic Auth Client Flags", pflag.ExitOnError)
		fs.StringP("username", "u", "", `Username for basic authentication.
If a password is not also provided then the client may enter any password.`)
		fs.StringP("password", "P", "", `Password for basic authentication.
If a username is not also provided using the --username flag then the client may enter any username.
If either the --password-prompt or --password-file flags are set, this flag will be ignored.`)
		fs.String("password-file", "", `Path to file containing password for basic authentication.
If a username is not also provided then the client may enter any username.
If the --password-prompt flag is set, this flags will be ignored.`)
		fs.BoolP("password-prompt", "W", false, `Prompt for password for basic authentication.
If a username is not also provided then the client may enter any username.
If the --password-file flag is set, this flag will be ignored.`)
		return fs
	})
}

func (c *BasicAuth) setFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
	cmd.MarkFlagsMutuallyExclusive("password", "password-file")
	cmd.MarkFlagFilename("password-file")
	cmd.MarkFlagFilename("unauthorized-page")
}

func (c *BasicAuth) mergeFlags() {
	if c.fs.Changed("username") {
		c.Username, _ = c.fs.GetString("username")
	}
	if c.fs.Changed("password") {
		c.Password, _ = c.fs.GetString("password")
	}
	if c.fs.Changed("password-file") {
		c.PasswordFile, _ = c.fs.GetString("password-file")
	}
	if c.fs.Changed("password-prompt") {
		c.PasswordPrompt, _ = c.fs.GetBool("password-prompt")
	}
	if c.fs.Changed("unauthorized-page") {
		c.UnauthorizedPage, _ = c.fs.GetString("unauthorized-page")
	}
	if c.fs.Changed("unauthorized-status") {
		c.UnauthorizedStatus, _ = c.fs.GetInt("unauthorized-status")
	}
	if c.fs.Changed("no-dialog") {
		c.NoDialog, _ = c.fs.GetBool("no-dialog")
	}
}

func (c *BasicAuth) validate() error {
	return nil
}
