package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type FilesMap map[string][]byte

func (fm FilesMap) projectInto(dir string) error {
	for path, contents := range fm {
		var (
			path      = filepath.Join(dir, path)
			parentDir = filepath.Dir(path)
		)
		if err := os.MkdirAll(parentDir, 0700); err != nil {
			return err
		}

		if err := os.WriteFile(path, contents, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

func EOFReader(data []byte) io.Reader {
	return &stdinPayload{data: data}
}

type stdinPayload struct {
	data []byte

	r *io.PipeReader
	w *io.PipeWriter
}

func (sp *stdinPayload) Read(p []byte) (int, error) {
	if sp.r == nil || sp.w == nil {
		sp.r, sp.w = io.Pipe()
		go func() {
			sp.w.Write(sp.data)
			sp.w.Close()
		}()
	}

	return sp.r.Read(p)
}

type retryClient struct {
	client http.RoundTripper
}

func (rc *retryClient) post(url, mime string, body io.Reader) *http.Response {
	var response *http.Response

	if rc.client == nil {
		rc.client = &http.Transport{}
	}

	for response == nil {
		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			panic(fmt.Sprintf("invalid url: %v", err))
		}
		req.Header.Set("Content-Type", mime)
		response, _ = rc.client.RoundTrip(req)
	}

	return response
}

func (rc *retryClient) get(url string) (*http.Response, error) {
	var response *http.Response

	if rc.client == nil {
		rc.client = &http.Transport{}
	}

	for response == nil {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		response, _ = rc.client.RoundTrip(req)
		time.Sleep(10 * time.Millisecond)
	}

	return response, nil
}
