package conf

import (
	"time"
)

var archiveMethodDefault = "tar.gz"
var shellDefault = "/bin/sh"
var noUnixNormDefault = false

type Conf struct {
	NoInfo     bool
	NoError    bool
	Port       string
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

	ReplaceHeaders bool     // only valid if cgi is true or shellCommand != ""
	EnvVars        []string // only used if cgi is true or shellCommand != ""
	CgiStderr      string   // only used if cgi is true or shellCommand != ""
	Dir            string   // only used if cgi is true or shellCommand != ""

	NoUnixNorm bool
}

func NewConf() *Conf {
	c := &Conf{
		ArchiveMethod: archiveMethodDefault,
		Shell:         shellDefault,
		RawHeaders:    []string{},
		EnvVars:       []string{},
	}
	return c
}
