package receive

import (
	"errors"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jf-tech/iohelper"
	"github.com/raphaelreyna/oneshot/v2/internal/commands"
	"github.com/raphaelreyna/oneshot/v2/internal/file"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/internal/net/http"
	"github.com/raphaelreyna/oneshot/v2/internal/out"
	"github.com/spf13/cobra"
)

func New() *Cmd {
	return &Cmd{
		header: make(http.Header),
	}
}

type Cmd struct {
	file                 *file.FileWriter
	writeTemplate        func(io.Writer, bool) error
	cobraCommand         *cobra.Command
	header               http.Header
	csrfToken            string
	unixEOLNormalization bool
	decodeBase64Output   bool
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.header == nil {
		c.header = make(http.Header)
	}
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:  "receive [dir]",
		RunE: c.setServer,
	}

	flags := c.cobraCommand.Flags()
	flags.String("csrf-token", "", "Use a CSRF token, if left empty, a random one will be generated")
	flags.String("eol", "unix", "How to parse EOLs in the received file. 'unix': '\\n', 'dos': '\\r\\n' ")
	flags.StringP("ui", "U", "", "Name of ui file to use")
	flags.Bool("decode-b64", false, "Decode base-64")

	return c.cobraCommand
}

func (c *Cmd) setServer(cmd *cobra.Command, args []string) error {
	var (
		ctx            = cmd.Context()
		flags          = cmd.Flags()
		headerSlice, _ = flags.GetStringSlice("header")
		eol, _         = flags.GetString("eol")

		err error
	)
	c.decodeBase64Output, _ = flags.GetBool("decode-b64")
	c.csrfToken, _ = flags.GetString("csrf-token")

	c.unixEOLNormalization = eol == "unix"
	c.header = oneshothttp.HeaderFromStringSlice(headerSlice)
	c.file = &file.FileWriter{}

	var (
		// if no args were passed, we can assume the user wants to receive to stdout
		writingTostdout = len(args) == 0
		fileDirPath     string
	)

	// if writing to stdout
	if writingTostdout {
		// let the out package know
		out.ReceivingToStdout(ctx)
	} else {
		// otherwise grab the first arg as the file path directory the user wants to receive to
		fileDirPath, err = filepath.Abs(args[0])
		if err != nil {
			return err
		}

		// get info on the fileDirPath
		stat, err := os.Stat(fileDirPath)
		// if it doesnt exist
		if err != nil {
			// then the user wants to receive to a file that doesnt exist yet and has given
			// us the file and directory names all in 1 string
			dirName, fileName := filepath.Split(fileDirPath)
			c.file.SetUserProvidedName(fileName)
			c.file.Path = dirName
		} else {
			// otherwise the user
			if stat.IsDir() {
				// either wants to receive to an existing directory and
				// hasnt given us the name of the file they want to receive to
				// so we can only set the directory name
				c.file.Path = fileDirPath
			} else {
				// or the user wants to overwrite an existing file
				dirName, fileName := filepath.Split(fileDirPath)
				c.file.SetUserProvidedName(fileName)
				c.file.Path = dirName
			}
		}

		// make sure we can write to the directory we're receiving to
		if err = isDirWritable(c.file.Path); err != nil {
			return err
		}
	}

	var (
		tmpl = template.New("base")
		ui   = os.Getenv("ONESHOT_UI")
	)

	tmpl = tmpl.Funcs(template.FuncMap{
		"enableBase64Decoding": func() error {
			c.decodeBase64Output = true
			return nil
		},
	})

	if flagUI, _ := flags.GetString("ui"); flagUI != "" {
		ui = flagUI
	}

	if ui != "" {
		tmpl, err = tmpl.ParseGlob(ui)
		if err != nil {
			return err
		}
	} else {
		// create the writeTemplate func to execute the template into the RequestWriter.
		tmpl = template.New("internal")
		if tmpl, err = tmpl.Parse(receivePageFileSectionTemplate); err != nil {
			return err
		}
		if tmpl, err = tmpl.Parse(receivePageInputSectionTemplate); err != nil {
			return err
		}
		if tmpl, err = tmpl.Parse(receivePageBaseTemplate); err != nil {
			return err
		}
	}

	// execute template to run config funcs it may have set
	if ui != "" {
		_ = tmpl.ExecuteTemplate(io.Discard, "oneshot", nil)
	}

	sections := struct {
		FileSection  bool
		InputSection bool
		CSRFToken    string
		WithJS       bool
	}{
		FileSection:  true,
		InputSection: true,
		CSRFToken:    c.csrfToken,
	}
	c.writeTemplate = func(w io.Writer, withJS bool) error {
		sections.WithJS = withJS
		return tmpl.ExecuteTemplate(w, "oneshot", &sections)
	}

	commands.SetHTTPHandlerFunc(ctx, c.ServeHTTP)
	return nil
}

type httpError struct {
	error
	stat int
}

func (he *httpError) Unwrap() error {
	return he.error
}

func (c *Cmd) readCloserFromMultipartFormData(r *http.Request) (io.ReadCloser, int64, error) {
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
	clientProvidedName := fileName(cd)
	c.file.SetClientProvidedName(clientProvidedName)

	cl, _ := strconv.ParseInt(part.Header.Get("Content-Length"), 10, 64)
	// if we couldnt get the content length from a Content-Length header
	if cl == 0 {
		// try to get it from our own injected header
		if mpLengthsString := r.Header.Get("X-Oneshot-Multipart-Content-Lengths"); mpLengthsString != "" {
			mpls := strings.Split(mpLengthsString, ";")
			if 0 < len(mpls) {
				nameSize := strings.Split(mpls[0], "=")
				if len(nameSize) == 2 {
					if nameSize[0] == clientProvidedName {
						size, err := strconv.ParseInt(nameSize[1], 10, 64)
						if err == nil {
							cl = size
						}
					}
				}

			}
		}
	}

	return part, cl, nil
}

func (c *Cmd) readCloserFromApplicationWWWForm(r *http.Request) (io.ReadCloser, int64, error) {
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

	return io.NopCloser(src), 0, nil
}

func (c *Cmd) readCloserFromRawBody(r *http.Request) (io.ReadCloser, int64, error) {
	// Check for csrf token if we care to
	if c.csrfToken != "" && r.Header.Get("X-CSRF-Token") != c.csrfToken {
		return nil, 0, &httpError{
			error: errors.New("invalid CSRF token"),
			stat:  http.StatusUnauthorized,
		}
	}

	cd := r.Header.Get("Content-Disposition")
	fn := fileName(cd)
	c.file.SetClientProvidedName(fn)

	c.file.MIMEType = r.Header.Get("Content-Type")
	cl, _ := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)

	return r.Body, cl, nil
}
