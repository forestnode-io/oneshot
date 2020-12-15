package conf

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strings"
	"time"
)

const (
	DownloadMode uint8 = iota
	UploadMode
	CGIMode
	RedirectMode
)

var archiveMethodDefault = "tar.gz"
var shellDefault = "/bin/sh"
var noUnixNormDefault = false

type Conf struct {
	NoInfo     bool
	NoError    bool
	Port       string
	Host string
	Timeout    time.Duration
	NoDownload bool

	Upload      bool
	UploadFile  bool
	UploadInput bool

	ExitOnFail bool

	ArchiveMethod string

	Mdns bool

	FileName string
	FileExt  string
	FileMime string

	CertFile string
	KeyFile  string
	Sstls    bool

	Username       string
	Password       string
	PasswordFile   string
	PasswordHidden bool

	RawHeaders []string

	Cgi          bool
	CgiStrict    bool // only valid if cgi is true
	ShellCommand bool
	Shell        string // only used if shellCommand != ""

	Redirect bool
	RedirectStatus int

	ReplaceHeaders bool     // only valid if cgi is true or shellCommand != ""
	EnvVars        []string // only used if cgi is true or shellCommand != ""
	CgiStderr      string   // only used if cgi is true or shellCommand != ""
	Dir            string   // only used if cgi is true or shellCommand != ""

	NoUnixNorm bool

	WaitForEOF bool

	AllowBots bool

	cmdFlagSet *pflag.FlagSet

	sstlsLoc    string
	credFileLoc string

	randUser bool
	randPass bool

	stdinBufLoc string //location of tempfile
}

func NewConf(cmd *cobra.Command) *Conf {
	c := &Conf{
		ArchiveMethod: archiveMethodDefault,
		Shell:         shellDefault,
		RawHeaders:    []string{},
		EnvVars:       []string{},

		cmdFlagSet: cmd.Flags(),
	}
	return c
}

func (c *Conf) Parse() error {
	// Set up web auth
	err := c.SetupCredentials()
	if err != nil {
		return err
	}

	// Clean up the port string
	c.Port = strings.ReplaceAll(c.Port, ":", "")

	// Grab location of self signed tls cert
	c.sstlsLoc, err = c.SetupCertAndKey(c.cmdFlagSet)
	if err != nil {
		return err
	}
	return nil
}

// SSTLSLoc returns an empty string and false if self signed tls cert is not being used
func (c *Conf) SSTLSLoc() (string, bool) {
	return c.sstlsLoc, c.sstlsLoc != ""
}

// CredFileLoc returns an empty string and false if randomly generated were never written to disk
func (c *Conf) CredFileLoc() (string, bool) {
	return c.credFileLoc, c.credFileLoc != ""
}

func (c *Conf) RandCredentials() (user, password bool) {
	return c.randUser, c.randPass
}

func (c *Conf) Mode() uint8 {
	switch {
	case c.Redirect:
		return RedirectMode
	case c.Upload || c.UploadFile || c.UploadInput:
		return UploadMode
	case c.Cgi || c.CgiStrict || c.ShellCommand:
		return CGIMode
	}
	return DownloadMode
}

func (c *Conf) StdinBufferLocation() string {
	return c.stdinBufLoc
}
