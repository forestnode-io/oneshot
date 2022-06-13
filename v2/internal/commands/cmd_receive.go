package commands

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/user"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jf-tech/iohelper"
	"github.com/raphaelreyna/oneshot/v2/internal/file"
	"github.com/raphaelreyna/oneshot/v2/internal/server"
	"github.com/raphaelreyna/oneshot/v2/internal/stdout"
	"github.com/raphaelreyna/oneshot/v2/internal/summary"
	"github.com/spf13/cobra"
)

func init() {
	x := receiveCmd{
		header: make(http.Header),
	}
	root.AddCommand(x.command())
}

type receiveCmd struct {
	file                 *file.FileWriter
	writeTemplate        func(io.Writer) error
	cobraCommand         *cobra.Command
	header               http.Header
	csrfToken            string
	unixEOLNormalization bool
}

func (c *receiveCmd) command() *cobra.Command {
	c.cobraCommand = &cobra.Command{
		Use:  "receive [dir]",
		RunE: c.runE,
	}

	flags := c.cobraCommand.Flags()
	flags.StringP("name", "n", "", "Name of the file")
	flags.String("csrf-token", "", "Use a CSRF token, if left empty, a random one will be generated")
	flags.String("eol", "unix", "How to parse EOLs in the received file. 'unix': '\\n', 'dos': '\\r\\n' ")

	return c.cobraCommand
}

func (c *receiveCmd) runE(cmd *cobra.Command, args []string) error {
	var (
		ctx            = cmd.Context()
		flags          = cmd.Flags()
		headerSlice, _ = flags.GetStringSlice("header")
		name, _        = flags.GetString("name")
		csrfToken, _   = flags.GetString("csrf-token")
		eol, _         = flags.GetString("eol")
	)

	if len(args) == 0 {
		stdout.ReceivingToStdout(ctx)
	}

	c.csrfToken = csrfToken
	c.unixEOLNormalization = eol == "unix"
	c.header = headerFromStringSlice(headerSlice)
	c.file = &file.FileWriter{}
	var fileDirPath string
	if 0 < len(args) {
		fileDirPath = args[0]
	}
	if name != "" {
		c.file.SetName(name, false)
		if fileDirPath == "" {
			// not writing to stdout, omitting a dir while supplying a name defaults to the cwd
			fileDirPath = "."
		}
	}
	c.file.ProgressWriter = os.Stdout
	if fileDirPath == "" {
		// writing file (and only the file) to stdout
		c.file.ProgressWriter = nil
	} else {
		if err := isDirWritable(fileDirPath); err != nil {
			return err
		}
		c.file.Path = fileDirPath
	}

	var err error
	tmpl := template.New("file-section")
	if tmpl, err = tmpl.Parse(receivePageFileSectionTemplate); err != nil {
		return err
	}
	if tmpl, err = tmpl.Parse(receivePageInputSectionTemplate); err != nil {
		return err
	}
	if tmpl, err = tmpl.Parse(receivePageBaseTemplate); err != nil {
		return err
	}
	sections := struct {
		FileSection  bool
		InputSection bool
		CSRFToken    string
	}{
		FileSection:  true,
		InputSection: true,
		CSRFToken:    c.csrfToken,
	}
	c.writeTemplate = func(w io.Writer) error {
		return tmpl.ExecuteTemplate(w, "base", &sections)
	}

	srvr := server.NewServer(c)
	setServer(ctx, srvr)
	return nil
}

func (c *receiveCmd) ServeHTTP(w http.ResponseWriter, r *http.Request) (*summary.Request, error) {
	if r.Method == "GET" {
		c._handleGET(w, r)
		return nil, nil
	}

	sr := summary.NewRequest(r)
	var (
		src io.Reader
		cl  int64 // content-length
		err error
	)
	// Switch on the type of upload to obtain the appropriate src io.Reader to read data from.
	// Uploads may happen by uploading a file, uploading text from an HTML text box, or straight from the request body
	rct := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(rct, "multipart/form-data"): // User uploaded a file
		reader, err := r.MultipartReader()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil, err
		}

		// Check for csrf token if we care to
		if c.csrfToken != "" {
			part, err := reader.NextPart()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return nil, err
			}

			if !strings.Contains(part.Header.Get("Content-Disposition"), "csrf-token") {
				err := errors.New("missing CRSF token")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return nil, err
			}

			partData, err := io.ReadAll(part)
			if err != nil {
				err := errors.New("unable to read CSRF token")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return nil, err
			}

			if string(partData) != c.csrfToken {
				err := errors.New("invalid CSRF token")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return nil, err
			}
		}

		part, err := reader.NextPart()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil, err
		}

		cd := part.Header.Get("Content-Disposition")
		if c.file.Path != "" && c.file.Name() == "" {
			if fn := fileName(cd); fn != "" {
				if c.file.Path == "" {
					c.file.Path = "."
				}
				c.file.SetName(fn, true)
			}
		}

		src = part

		cl, err = strconv.ParseInt(part.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			cl = 0
		}
	case strings.Contains(rct, "application/x-www-form-urlencoded"): // User uploaded text from HTML text box
		foundCSRFToken := false
		// Assume we found the CSRF token if the user doesn't care to use one
		if c.csrfToken == "" {
			foundCSRFToken = true
		}

		// Look for the CSRF token in the header
		if r.Header.Get("X-CSRF-Token") == c.csrfToken && c.csrfToken != "" {
			foundCSRFToken = true
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil, err
		}

		// If we havent found the CSRF token yet, look for it in the parsed form data
		if !foundCSRFToken && r.PostFormValue("csrf-token") != c.csrfToken {
			err := errors.New("invalid CSRF token")
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return nil, err
		}

		src = strings.NewReader(r.PostForm.Get("text"))
		if c.unixEOLNormalization {
			src = iohelper.NewBytesReplacingReader(src, crlf, lf)
		}
	default: // Could not determine how file upload was initiated, grabbing the request body
		// Check for csrf token if we care to
		if c.csrfToken != "" && r.Header.Get("X-CSRF-Token") != c.csrfToken {
			err := errors.New("invalid CSRF token")
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return nil, err
		}

		cd := r.Header.Get("Content-Disposition")
		if c.file.Path != "" && c.file.Name() == "" {
			if fn := fileName(cd); fn != "" {
				if c.file.Path == "" {
					c.file.Path = "."
				}
				c.file.SetName(fn, true)
			}
		}
		c.file.MIMEType = rct
		src = r.Body
		cl, err = strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			cl = 0
		}
	}

	c.file.Lock()
	if err == nil && cl != 0 {
		c.file.SetSize(cl)
	}
	if err = c.file.Open(); err != nil {
		c.file.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}

	defer r.Body.Close()
	_, err = io.Copy(c.file, src)
	if err != nil {
		c.file.Reset()
		c.file.Unlock()
		return nil, err
	}
	c.file.Unlock()

	return sr, nil
}

func (c *receiveCmd) ServeExpiredHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("expired hello from server"))
}

func (c *receiveCmd) _handleGET(w http.ResponseWriter, r *http.Request) {
	c.writeTemplate(w)
}

var (
	lf   = []byte{10}
	crlf = []byte{13, 10}
)

var regex = regexp.MustCompile(`filename="(.+)"`)

func fileName(s string) string {
	subs := regex.FindStringSubmatch(s)
	if len(subs) > 1 {
		return subs[1]
	}
	return ""
}

const (
	receivePageBaseTemplate = `{{ define "base" }}<!DOCTYPE html>
<html>
<head>
<link rel="apple-touch-icon" href="/assets/icon.png">
<link rel="icon" type="image/png" href="/assets/icon.png">
</head>
<body>
{{ if .FileSection }}
  {{ template "file-section" .CSRFToken }}
{{ end }}
{{ if .InputSection }}
  {{ if .FileSection }}
    <br/>OR<br/>
  {{ end }}
  {{ template "input-section" .CSRFToken }}
{{ end }}
</body>
</html>
{{ end }}`

	receivePageFileSectionTemplate = `{{ define "file-section" }}<form action="/" method="post" enctype="multipart/form-data">
  {{ if ne . "" }}<input type="hidden" name="csrf-token" value="{{ . }}">{{ end }}
  <h5>Select a file to upload:</h5>
  <input type="file" name="oneshot">
  <br><br>
  <input type="submit" value="Upload">
</form>{{ end }}`

	receivePageInputSectionTemplate = `{{ define "input-section" }}<form action="/" method="post">
  {{ if ne . "" }}<input type="hidden" name="csrf-token" value="{{ . }}">{{ end }}
  <h5>Enter text to send: </h5>
  <textarea name="text"></textarea>
  <br><br>
  <input type="submit" value="Upload">
</form>{{ end }}`
)

func isDirWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	if runtime.GOOS == "windows" {
		return _isDirWritable_windows(path, info)
	}
	return _isDirWritable_unix(path, info)
}

func _isDirWritable_windows(path string, info os.FileInfo) error {
	testFileName := fmt.Sprintf("oneshot%d", time.Now())
	file, err := os.Create(testFileName)
	if err != nil {
		return err
	}
	file.Close()
	os.Remove(file.Name())
	return nil
}

func _isDirWritable_unix(path string, info os.FileInfo) error {
	const (
		bmOthers = 0x0002
		bmGroup  = 0x0020
		bmOwner  = 0x0200
	)
	var mode = info.Mode()

	// check if writable by others
	if mode&bmOthers != 0 {
		return nil
	}

	stat := info.Sys().(*syscall.Stat_t)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	// check if writable by group
	if mode&bmGroup != 0 {
		gid := fmt.Sprint(stat.Gid)
		gids, err := usr.GroupIds()
		if err != nil {
			return err
		}
		for _, g := range gids {
			if g == gid {
				return nil
			}
		}
	}

	// check if writable by owner
	if mode&bmOwner != 0 {
		uid := fmt.Sprint(stat.Uid)
		if uid == usr.Uid {
			return nil
		}
	}

	return fmt.Errorf("%s: permission denied", path)
}
