package cgi

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var DefaultOutputHandler OutputHandler = func(w http.ResponseWriter, r *http.Request,
	h *Handler, stdoutRead io.Reader) {
	var (
		linebody     = bufio.NewReaderSize(stdoutRead, 1024)
		headers      = make(http.Header)
		statusCode   = 0
		headerLines  = 0
		sawBlankLine = false
	)

	for {
		line, isPrefix, err := linebody.ReadLine()
		if isPrefix {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(h.stderr, "cgi: long header line from subprocess.\n")
			return
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(h.stderr, "cgi: error reading headers: %v\n", err)
			return
		}
		if len(line) == 0 {
			sawBlankLine = true
			break
		}
		headerLines++
		parts := strings.SplitN(string(line), ":", 2)
		if len(parts) < 2 {
			fmt.Fprintf(h.stderr, "cgi: bogus header line: %s\n", string(line))
			continue
		}
		header, val := parts[0], parts[1]
		header = strings.TrimSpace(header)
		val = strings.TrimSpace(val)
		switch {
		case header == "Status":
			if len(val) < 3 {
				fmt.Fprintf(h.stderr, "cgi: bogus status (short): %q\n", val)
				return
			}
			code, err := strconv.Atoi(val[0:3])
			if err != nil {
				fmt.Fprintf(h.stderr, "cgi: bogus status: %q\n", val)
				fmt.Fprintf(h.stderr, "cgi: line was %q\n", line)
				return
			}
			statusCode = code
		default:
			headers.Add(header, val)
		}
	}
	if headerLines == 0 || !sawBlankLine {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(h.stderr, "cgi: no headers\n")
		return
	}

	if loc := headers.Get("Location"); loc != "" {
		if statusCode == 0 {
			statusCode = http.StatusFound
		}
	}

	if statusCode == 0 && headers.Get("Content-Type") == "" {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(h.stderr, "cgi: missing required Content-Type in headers\n")
		return
	}

	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	for k, vv := range headers {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(statusCode)

	_, err := io.Copy(w, linebody)
	if err != nil {
		fmt.Fprintf(h.stderr, "cgi: copy error: %v\n", err)
	}
}

var portRegex = regexp.MustCompile(`:([0-9]+)$`)

func upperCaseAndUnderscore(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z':
		return r - ('a' - 'A')
	case r == '-':
		return '_'
	case r == '=':
		return '_'
	}
	return r
}

// EZOutputHandler sends the entire output of the client process without scanning for headers.
// Always responds with a 200 status code.
var EZOutputHandler OutputHandler = func(w http.ResponseWriter, r *http.Request, h *Handler, stdoutRead io.Reader) {
	for k, vv := range h.header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(http.StatusOK)

	linebody := bufio.NewReaderSize(stdoutRead, 1024)
	_, err := io.Copy(w, linebody)
	if err != nil {
		fmt.Fprintf(h.stderr, "cgi: copy error: %v", err)
		return
	}
}

// OutputHandlerReplacer scans the output of the client process for headers which replaces the default header values.
// Stops scanning for headers after encountering the first non-header line.
// The rest of the output is then sent as the response body.
var OutputHandlerReplacer OutputHandler = func(w http.ResponseWriter, r *http.Request, h *Handler, stdoutRead io.Reader) {
	internalError := func(err error) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(h.stderr, "CGI error: %v", err)
	}

	// readBytes holds the bytes read during header scan but that aren't part of the header.
	// This data will be added to the front of the responses body
	var readBytes []byte
	linebody := bufio.NewReaderSize(stdoutRead, 1024)
	statusCode := 0

	for {
		line, tooBig, err := linebody.ReadLine()
		if tooBig || err == io.EOF {
			break
		}
		if err != nil {
			internalError(err)
			return
		}
		if len(line) == 0 {
			break
		}

		parts := strings.SplitN(string(line), ":", 2)
		if len(parts) < 2 {
			// This line is not a header, add it to the head of the body and break
			readBytes = append(line, '\n')
			break
		}

		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])

		switch {
		case k == "Status":
			if len(v) < 3 {
				fmt.Fprintf(h.stderr, "cgi: bogus status (short): %q\n", v)
				return
			}
			code, err := strconv.Atoi(v[0:3])
			if err != nil {
				fmt.Fprintf(h.stderr, "cgi: bogus status: %q\n", v)
				fmt.Fprintf(h.stderr, "cgi: line was %q\n", line)
				return
			}
			statusCode = code
		default:
			h.header.Set(k, v)
		}
	}
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	for k, vv := range h.header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(statusCode)

	// Add back in the beginning portion of the body that was slurped up while scanning for headers.
	if readBytes != nil {
		_, err := w.Write(readBytes)
		if err != nil {
			fmt.Fprintf(h.stderr, "cgi: copy error: %v\n", err)
			return
		}
	}

	_, err := io.Copy(w, linebody)
	if err != nil {
		fmt.Fprintf(h.stderr, "cgi: copy error: %v\n", err)
		return
	}
}
