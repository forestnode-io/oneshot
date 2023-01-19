package itest

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

type Oneshot struct {
	T          *testing.T
	Env        []string
	Args       []string
	Files      FilesMap
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	WorkingDir string

	Cmd       *exec.Cmd
	stdoutBuf *bytes.Buffer
	stderrBuf *bytes.Buffer
}

func (o *Oneshot) Cleanup() {}

func (o *Oneshot) Start() {
	if o.Files != nil {
		o.Files.ProjectInto(o.WorkingDir)
	}

	if o.Stdout == nil {
		o.stdoutBuf = bytes.NewBuffer(nil)
		o.Stdout = o.stdoutBuf
	}

	if o.Stderr == nil {
		o.stderrBuf = bytes.NewBuffer(nil)
		o.Stderr = o.stderrBuf
	}

	if o.Cmd == nil {
		o.Cmd = exec.Command(
			filepath.Join(o.WorkingDir, "../../oneshot.testing"),
			o.Args...,
		)
		o.Cmd.Stdin = o.Stdin
		o.Cmd.Stdout = o.Stdout
		o.Cmd.Stderr = o.Stderr
		o.Cmd.Dir = o.WorkingDir
		o.Cmd.Env = append(os.Environ(), o.Env...)
	}

	if err := o.Cmd.Start(); err != nil {
		o.T.Fatalf("unable to start oneshot exec: %v\n", err)
	}

	time.Sleep(time.Second)
}

func (o *Oneshot) Wait() {
	if o.Cmd == nil {
		o.T.Fatal("attempting to exit oneshot but oneshot it not running")
	}
	o.Cmd.Wait()
}

func (o *Oneshot) Signal(sig os.Signal) {
	if o.Cmd != nil {
		o.Cmd.Process.Signal(sig)
	}
}
