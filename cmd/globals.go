package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"time"
)

const (
	downloadMode uint8 = iota
	uploadMode
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

	upload bool

	exitOnFail bool

	archiveMethod string

	mdns bool

	fileName string
	fileExt  string
	fileMime string

	certFile string
	keyFile  string
	sstls    bool

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
	Use:     "oneshot [flags]... [file|dir]",
	Version: fmt.Sprintf(": %s\ndate: %s\nauthor: Raphael Reyna\n", version, date),
	Short:   "A single-fire first-come-first-serve HTTP server.",
	Long: `Transfer files and data easily between your computer and any browser or HTTP client.
The first client to connect is given the file or uploads a file, all others receive an HTTP 410 Gone response code.
Directories will automatically be archived before being sent (see -a, --archive-method for more information).
`,
	Run: run,
}
