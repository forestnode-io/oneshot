package cgi

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type OutputHandler func(w http.ResponseWriter, r *http.Request, h *Handler, stdoutReader io.Reader)

type HandlerConfig struct {
	Cmd         []string
	WorkingDir  string
	InheritEnvs []string
	BaseEnv     []string
	Header      http.Header

	OutputHandler OutputHandler
	Stderr        io.Writer
}

type Handler struct {
	execPath   string
	args       []string
	workingDir string

	env    []string
	header http.Header

	outputHandler OutputHandler

	stderr io.Writer
}

func NewHandler(conf HandlerConfig) (*Handler, error) {
	var (
		l = len(conf.Cmd)
		h = Handler{
			workingDir:    conf.WorkingDir,
			header:        conf.Header,
			env:           NewEnv(conf.BaseEnv, conf.InheritEnvs),
			outputHandler: conf.OutputHandler,
			stderr:        conf.Stderr,
		}
	)

	if l == 0 {
		return nil, errors.New("command required")
	}
	h.execPath = findExec(conf.Cmd[0])

	if 1 < l {
		h.args = conf.Cmd[1:]
	}

	if h.header == nil {
		h.header = http.Header{
			"Content-Type": []string{"text/plain"},
		}
	}

	if h.workingDir == "" {
		var err error
		if h.workingDir, err = os.Getwd(); err != nil {
			return nil, err
		}
	}

	if h.outputHandler == nil {
		h.outputHandler = DefaultOutputHandler
	}

	if h.stderr == nil {
		h.stderr = io.Discard
	}

	return &h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(r.TransferEncoding) > 0 && r.TransferEncoding[0] == "chunked" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Chunked request bodies are not supported by CGI."))
		return
	}

	internalError := func(err error) {
		fmt.Fprintf(h.stderr, "internal server error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	cmd := &exec.Cmd{
		Path:   h.execPath,
		Args:   append([]string{h.execPath}, h.args...),
		Dir:    h.workingDir,
		Env:    AddRequest(h.env, r),
		Stderr: h.stderr,
	}

	if r.ContentLength != 0 {
		cmd.Stdin = r.Body
	}
	stdoutRead, err := cmd.StdoutPipe()
	if err != nil {
		internalError(fmt.Errorf("unable to get cmd stdout pipe: %w", err))
		return
	}
	err = cmd.Start()
	if err != nil {
		internalError(fmt.Errorf("unable to get start cmd: %w", err))
		return
	}

	defer func() {
		if err := cmd.Wait(); err != nil {
			internalError(fmt.Errorf("cmd failed %s: %w", cmd.Path+strings.Join(cmd.Args, " "), err))
		}
	}()
	defer stdoutRead.Close()

	h.outputHandler(w, r, h, stdoutRead)

	// Make sure the process is good and dead before exiting
	if err := cmd.Process.Kill(); err != nil {
		log.Printf("unable to kill process %d: %v", cmd.Process.Pid, err)
	}
}
