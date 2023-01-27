package receive

import (
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jf-tech/iohelper"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	"github.com/raphaelreyna/oneshot/v2/pkg/file"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/spf13/cobra"
)

func New() *Cmd {
	return &Cmd{
		header: make(http.Header),
	}
}

type Cmd struct {
	fileTransferConfig   *file.WriteTransferConfig
	writeTemplate        func(io.Writer, bool) error
	cobraCommand         *cobra.Command
	header               http.Header
	csrfToken            string
	unixEOLNormalization bool
	decodeBase64Output   bool
	statusCode           int
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.header == nil {
		c.header = make(http.Header)
	}
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "receive [file]",
		RunE:  c.setHandlerFunc,
		Short: "Receive a file from the client",
		Long: `Receive a file from the client. If file is not specified, the content will be sent to stdout.

A simple web interface is served by default to GET requests to allow users to upload files from their browser.
Users are not required to use the web interface to upload files, they can use any HTTP client to upload files to the server via a POST request.
Uploading files via the web interface can be restricted to a specific CSRF token by using the --csrf-token flag.

The web interface can be customized by providing a ui file or path glob via the --ui flag.
These files are parsed as Go templates.
The CSRF token (if one is being used) can be accessed through the template variable ` + "`.CSRFToken`" + `

A web interface template can also instruct oneshot to decode base64 encoded uploads by calling the ` + "`.enableBase64Decoding`" + ` function in the template.
This is useful for web interfaces that upload files that are base64 encoded such as an image or video.
These encoded files will be saved decoded to disk.

In order to display dynamic transfer information, oneshot needs to know the total size of the file being uploaded.
Web interfaces can provide this information by setting the Content-Length header on the POST request.
If a file is being uploaded as a multipart form, the content length can be provided by setting the ` + "`X-Oneshot-Multipart-Content-Lengths`" + ` header in the request.
Values in the ` + "`X-Oneshot-Multipart-Content-Lengths`" + ` header should be of the form <FILE NAME>=<CONTENT LENGTH>.
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("too many arguments")
			}

			return nil
		},
	}

	flags := c.cobraCommand.Flags()
	flags.String("csrf-token", "", "Use a CSRF token, if left empty, a random one will be generated.")

	flags.String("eol", "unix", `How to parse EOLs in the received file.
Acceptable values are 'unix' and 'dos'; 'unix': '\n', 'dos': '\r\n'.`)

	flags.StringP("ui", "U", "", "Name of ui file to use.")

	flags.Bool("decode-b64", false, "Decode base-64.")

	flags.Int("status-code", 200, "HTTP status code sent to client.")

	flags.StringSliceP("header", "H", nil, `Header to send to client. Can be specified multiple times. 
Format: <HEADER NAME>=<HEADER VALUE>`)

	return c.cobraCommand
}

func (c *Cmd) setHandlerFunc(cmd *cobra.Command, args []string) error {
	var (
		ctx            = cmd.Context()
		flags          = cmd.Flags()
		headerSlice, _ = flags.GetStringSlice("header")
		eol, _         = flags.GetString("eol")

		err error
	)

	c.statusCode, _ = flags.GetInt("status-code")
	c.decodeBase64Output, _ = flags.GetBool("decode-b64")
	c.csrfToken, _ = flags.GetString("csrf-token")
	c.unixEOLNormalization = eol == "unix"
	c.header = oneshothttp.HeaderFromStringSlice(headerSlice)
	var location string
	if 0 < len(args) {
		location = args[0]
	}
	c.fileTransferConfig, err = file.NewWriteTransferConfig(ctx, location)
	if err != nil {
		return err
	}

	var (
		tmpl = template.New("base")
		ui   = ""
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
		tmpl = template.New("pkg")
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
		if err := tmpl.ExecuteTemplate(io.Discard, "oneshot", nil); err != nil {
			log.Printf("error during initial template execution (running config funcs): %s", err.Error())
		}
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

type requestBody struct {
	r    io.ReadCloser
	name string
	mime string
	size int64
}

func (c *Cmd) readCloserFromMultipartFormData(r *http.Request) (*requestBody, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, &httpError{
			error: err,
			stat:  http.StatusBadRequest,
		}
	}

	// Check for csrf token if we care to
	if c.csrfToken != "" {
		part, err := reader.NextPart()
		if err != nil {
			return nil, &httpError{
				error: err,
				stat:  http.StatusBadRequest,
			}
		}

		if !strings.Contains(part.Header.Get("Content-Disposition"), "csrf-token") {
			return nil, &httpError{
				error: errors.New("missing CRSF token"),
				stat:  http.StatusUnauthorized,
			}
		}

		partData, err := io.ReadAll(part)
		if err != nil {
			return nil, &httpError{
				error: errors.New("unable to read CSRF token"),
				stat:  http.StatusUnauthorized,
			}
		}

		if string(partData) != c.csrfToken {
			return nil, &httpError{
				error: errors.New("invalid CSRF token"),
				stat:  http.StatusUnauthorized,
			}
		}
	}

	part, err := reader.NextPart()
	if err != nil {
		return nil, &httpError{
			error: err,
			stat:  http.StatusBadRequest,
		}
	}

	cd := part.Header.Get("Content-Disposition")
	clientProvidedName := fileName(cd)

	contentLength, _ := strconv.ParseInt(part.Header.Get("Content-Length"), 10, 64)
	// if we couldnt get the content length from a Content-Length header
	if contentLength == 0 {
		// try to get it from our own injected header
		if mpLengthsString := r.Header.Get("X-Oneshot-Multipart-Content-Lengths"); mpLengthsString != "" {
			mpls := strings.Split(mpLengthsString, ";")
			if 0 < len(mpls) {
				nameSize := strings.Split(mpls[0], "=")
				if len(nameSize) == 2 {
					if nameSize[0] == clientProvidedName {
						size, err := strconv.ParseInt(nameSize[1], 10, 64)
						if err == nil {
							contentLength = size
						}
					}
				}

			}
		}
	}

	return &requestBody{
		r:    part,
		name: fileName(cd),
		mime: part.Header.Get("Content-Type"),
		size: contentLength,
	}, nil
}

func (c *Cmd) readCloserFromApplicationWWWForm(r *http.Request) (*requestBody, error) {
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
		return nil, &httpError{
			error: err,
			stat:  http.StatusBadRequest,
		}
	}

	// If we havent found the CSRF token yet, look for it in the parsed form data
	if !foundCSRFToken && r.PostFormValue("csrf-token") != c.csrfToken {
		return nil, &httpError{
			error: errors.New("invalid CSRF token"),
			stat:  http.StatusUnauthorized,
		}
	}

	var src io.Reader = strings.NewReader(r.PostForm.Get("text"))
	if c.unixEOLNormalization {
		src = iohelper.NewBytesReplacingReader(src, crlf, lf)
	}

	return &requestBody{
		r: io.NopCloser(src),
	}, nil
}

func (c *Cmd) readCloserFromRawBody(r *http.Request) (*requestBody, error) {
	// Check for csrf token if we care to
	if c.csrfToken != "" && r.Header.Get("X-CSRF-Token") != c.csrfToken {
		return nil, &httpError{
			error: errors.New("invalid CSRF token"),
			stat:  http.StatusUnauthorized,
		}
	}

	cd := r.Header.Get("Content-Disposition")
	contentLength, _ := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)

	return &requestBody{
		r:    r.Body,
		name: fileName(cd),
		size: contentLength,
		mime: r.Header.Get("Content-Type"),
	}, nil
}
