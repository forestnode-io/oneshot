package main

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
	Args       []string
	Files      FilesMap
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	WorkingDir string

	cmd       *exec.Cmd
	stdoutBuf *bytes.Buffer
	stderrBuf *bytes.Buffer
}

func (o *Oneshot) Cleanup() {}

func (o *Oneshot) Start() {
	if o.Files != nil {
		o.Files.projectInto(o.WorkingDir)
	}

	if o.Stdout == nil {
		o.stdoutBuf = bytes.NewBuffer(nil)
		o.Stdout = o.stdoutBuf
	}

	if o.Stderr == nil {
		o.stderrBuf = bytes.NewBuffer(nil)
		o.Stderr = o.stderrBuf
	}

	if o.cmd == nil {
		o.cmd = exec.Command(
			filepath.Join(o.WorkingDir, "../oneshot.testing"),
			o.Args...,
		)
		o.cmd.Stdin = o.Stdin
		o.cmd.Stdout = o.Stdout
		o.cmd.Stderr = o.Stderr
		o.cmd.Dir = o.WorkingDir
	}

	if err := o.cmd.Start(); err != nil {
		o.T.Fatalf("unable to start oneshot exec: %v\n", err)
	}

	time.Sleep(time.Second)
}

func (o *Oneshot) Wait() {
	if o.cmd == nil {
		o.T.Fatal("attempting to exit oneshot but oneshot it not running")
	}
	o.cmd.Wait()
}

func (o *Oneshot) Signal(sig os.Signal) {
	if o.cmd != nil {
		o.cmd.Process.Signal(sig)
	}
}
