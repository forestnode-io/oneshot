package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"time"
)

const (
	downloadMode uint8 = iota
	cgiMode
)

var mode uint8

var version string
var versionFlag bool
var date string

var (
	noInfo     bool
	noError    bool
	port       string
	timeout    time.Duration
	noDownload bool

	exitOnFail bool

	fileName string
	fileExt  string
	fileMime string

	certFile string
	keyFile  string

	username       string
	password       string
	passwordFile   string
	passwordHidden bool

	rawHeaders []string

	cgi            bool
	cgiStrict      bool // only valid if cgi is true
	shellCommand   bool
	shell          string   // only used if shellCommand != ""
	replaceHeaders bool     // only valid if cgi is true or shellCommand != ""
	envVars        []string // only used if cgi is true or shellCommand != ""
	cgiStderr      string   // only used if cgi is true or shellCommand != ""
	dir            string   // only used if cgi is true or shellCommand != ""
)

var RootCmd = &cobra.Command{
	Use:     "oneshot [flags]... [file]",
	Version: fmt.Sprintf(": %s\ndate: %s\nauthor: Raphael Reyna\n", version, date),
	Short:   "A single-fire HTTP server.",
	Long: `Start an HTTP server which will only serve files once.
The first client to connect is given the file, all others receive an HTTP 410 Gone response code.

If no file is given, oneshot will instead serve from stdin and hold the clients connection until receiving the EOF character.
`,
	Run: run,
}
