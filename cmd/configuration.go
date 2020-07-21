package cmd

import (
	"github.com/raphaelreyna/oneshot/pkg/server"
	"time"
)

//var archiveMethodDefault = "tar.gz"
//var shellDefault = "/bin/sh"

type conf struct {
	noInfo     *bool
	noError    *bool
	port       *string
	timeout    *time.Duration
	noDownload *bool

	upload      *bool
	uploadFile  *bool
	uploadInput *bool

	exitOnFail *bool

	archiveMethod *string

	mdns *bool

	fileName *string
	fileExt  *string
	fileMime *string

	certFile *string
	keyFile  *string
	sstls    *bool

	username       *string
	password       *string
	passwordFile   *string
	passwordHidden *bool

	rawHeaders []string

	cgi          *bool
	cgiStrict    *bool // only valid if cgi is true
	shellCommand *bool
	shell        *string // only used if shellCommand != ""

	replaceHeaders *bool    // only valid if cgi is true or shellCommand != ""
	envVars        []string // only used if cgi is true or shellCommand != ""
	cgiStderr      *string  // only used if cgi is true or shellCommand != ""
	dir            *string  // only used if cgi is true or shellCommand != ""

	noUnixNorm        *bool
	noUnixNormDefault *bool
}

func newConf() *conf {
	c := &conf{
		archiveMethod: &archiveMethodDefault,
		shell:         &shellDefault,
	}
	return c
}

func (c *conf) newServer() (*server.Server, error) {
	srvr := server.NewServer()
}
