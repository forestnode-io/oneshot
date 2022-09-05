package receive

import (
	"errors"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/jf-tech/iohelper"
	"github.com/raphaelreyna/oneshot/v2/internal/commands/shared"
	"github.com/raphaelreyna/oneshot/v2/internal/file"
	"github.com/raphaelreyna/oneshot/v2/internal/server"
	"github.com/raphaelreyna/oneshot/v2/internal/stdout"
	"github.com/spf13/cobra"
)

func New() *Cmd {
	return &Cmd{
		header: make(http.Header),
	}
}

type Cmd struct {
	file                 *file.FileWriter
	writeTemplate        func(io.Writer) error
	cobraCommand         *cobra.Command
	header               http.Header
	csrfToken            string
	unixEOLNormalization bool
}

func (c *Cmd) Cobra() *cobra.Command {
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

func (c *Cmd) runE(cmd *cobra.Command, args []string) error {
	var (
		ctx            = cmd.Context()
		flags          = cmd.Flags()
		headerSlice, _ = flags.GetStringSlice("header")
		name, _        = flags.GetString("name")
		csrfToken, _   = flags.GetString("csrf-token")
		eol, _         = flags.GetString("eol")
		_, wantsJSON   = flags.GetString("json")
	)

	if len(args) == 0 {
		stdout.ReceivingToStdout(ctx)
	}

	c.csrfToken = csrfToken
	c.unixEOLNormalization = eol == "unix"
	c.header = shared.HeaderFromStringSlice(headerSlice)
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

	c.file.ProgressWriter = nil
	if wantsJSON == nil {
		c.file.ProgressWriter = nil
	}

	if fileDirPath == "" {
		// writing file (and only the file) to stdout
		c.file.ProgressWriter = nil
	} else {
		if err := isDirWritable(fileDirPath); err != nil {
			return err
		}
		c.file.Path = fileDirPath
	}

	// create the writeTemplate func to execute the template into the RequestWriter.
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

	srvr := server.NewServer(c.ServeHTTP, c.ServeExpiredHTTP)
	server.SetServer(ctx, srvr)
	return nil
}

type httpError struct {
	error
	stat int
}

func (he *httpError) Unwrap() error {
	return he.error
}

func (c *Cmd) readerFromMultipartFormData(r *http.Request) (io.Reader, int64, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, 0, &httpError{
			error: err,
			stat:  http.StatusBadRequest,
		}
	}

	// Check for csrf token if we care to
	if c.csrfToken != "" {
		part, err := reader.NextPart()
		if err != nil {
			return nil, 0, &httpError{
				error: err,
				stat:  http.StatusBadRequest,
			}
		}

		if !strings.Contains(part.Header.Get("Content-Disposition"), "csrf-token") {
			return nil, 0, &httpError{
				error: errors.New("missing CRSF token"),
				stat:  http.StatusUnauthorized,
			}
		}

		partData, err := io.ReadAll(part)
		if err != nil {
			return nil, 0, &httpError{
				error: errors.New("unable to read CSRF token"),
				stat:  http.StatusUnauthorized,
			}
		}

		if string(partData) != c.csrfToken {
			return nil, 0, &httpError{
				error: errors.New("invalid CSRF token"),
				stat:  http.StatusUnauthorized,
			}
		}
	}

	part, err := reader.NextPart()
	if err != nil {
		return nil, 0, &httpError{
			error: err,
			stat:  http.StatusBadRequest,
		}
	}

	cd := part.Header.Get("Content-Disposition")
	if c.file.Name() == "" {
		if fn := fileName(cd); fn != "" {
			c.file.SetName(fn, true)
		}
	}

	cl, _ := strconv.ParseInt(part.Header.Get("Content-Length"), 10, 64)

	return part, cl, nil
}

func (c *Cmd) readerFromApplicationWWWForm(r *http.Request) (io.Reader, int64, error) {
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
		return nil, 0, &httpError{
			error: err,
			stat:  http.StatusBadRequest,
		}
	}

	// If we havent found the CSRF token yet, look for it in the parsed form data
	if !foundCSRFToken && r.PostFormValue("csrf-token") != c.csrfToken {
		return nil, 0, &httpError{
			error: errors.New("invalid CSRF token"),
			stat:  http.StatusUnauthorized,
		}
	}

	var src io.Reader = strings.NewReader(r.PostForm.Get("text"))
	if c.unixEOLNormalization {
		src = iohelper.NewBytesReplacingReader(src, crlf, lf)
	}

	return src, 0, nil
}

func (c *Cmd) readerFromRawBody(r *http.Request) (io.Reader, int64, error) {
	// Check for csrf token if we care to
	if c.csrfToken != "" && r.Header.Get("X-CSRF-Token") != c.csrfToken {
		return nil, 0, &httpError{
			error: errors.New("invalid CSRF token"),
			stat:  http.StatusUnauthorized,
		}
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

	c.file.MIMEType = r.Header.Get("Content-Type")
	cl, _ := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)

	return r.Body, cl, nil
}
