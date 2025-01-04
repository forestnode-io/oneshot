package itest

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/forestnode-io/oneshot/v2/pkg/configuration"
	"gopkg.in/yaml.v3"
)

type PortPool struct {
	sync.Mutex
	start   int
	end     int
	current int
}

func (pr *PortPool) Get() string {
	pr.Lock()
	defer pr.Unlock()

	if pr.current == 0 {
		pr.current = pr.start
	}
	p := pr.current
	pr.current++
	if pr.current > pr.end {
		panic("no available ports")
	}
	return strconv.Itoa(p)
}

var oneshotPortPool = &PortPool{
	start: 8080,
	end:   65535,
}

type Oneshot struct {
	T             *testing.T
	Env           []string
	Args          []string
	Files         FilesMap
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	WorkingDir    string
	TempDir       string
	Port          string
	Configuration *configuration.Root

	Cmd       *exec.Cmd
	StdoutBuf *bytes.Buffer
	StderrBuf *bytes.Buffer
}

func (o *Oneshot) Cleanup() {}

func (o *Oneshot) Start() {
	if o.Files != nil {
		o.Files.ProjectInto(o.WorkingDir)
	}

	if o.Stdout == nil {
		o.StdoutBuf = bytes.NewBuffer(nil)
		o.Stdout = o.StdoutBuf
	}

	if o.Stderr == nil {
		o.StderrBuf = bytes.NewBuffer(nil)
		o.Stderr = o.StderrBuf
	}

	// find "port" in the args and replace the following arg with the port
	setPort := false
	for i, arg := range o.Args {
		if (arg == "--port" || arg == "-p") && i+1 < len(o.Args) {
			o.Args[i+1] = o.Port
			setPort = true
			break
		}
	}
	if !setPort {
		o.Args = append(o.Args, "--port", o.Port)
	}

	if o.Configuration != nil {
		var err error
		o.Configuration.Server.Port, err = strconv.Atoi(o.Port)
		if err != nil {
			o.T.Fatalf("error converting port to int: %s", err)
			return
		}
		configBytes, err := yaml.Marshal(o.Configuration)
		if err != nil {
			o.T.Fatalf("error marshalling configuration: %s", err)
			return
		}
		configFileName := filepath.Join(o.TempDir, "config.yaml")
		if err := os.WriteFile(configFileName, configBytes, 0600); err != nil {
			o.T.Fatalf("error writing configuration file: %s", err)
			return
		}
		o.Env = append(o.Env, "ONESHOT_CONFIG="+configFileName)
	}

	if o.Cmd == nil {
		o.Cmd = exec.Command(
			filepath.Join(o.WorkingDir, "../oneshot.testing"),
			o.Args...,
		)
		o.Cmd.Stdin = o.Stdin
		o.Cmd.Stdout = o.Stdout
		o.Cmd.Stderr = o.Stderr
		o.Cmd.Dir = o.WorkingDir
		o.Cmd.Env = append(os.Environ(), o.Env...)
	}

	if err := o.Cmd.Start(); err != nil {
		msg := "error starting oneshot exec: %s\n"
		msg += "cmd: %s\n"
		msg += "args: %v\n"
		msg += "env: %v\n"
		msg += "working dir: %s\n"

		cmdDir := filepath.Dir(o.Cmd.Path)
		entries, err := os.ReadDir(cmdDir)
		if err != nil {
			msg += "error reading directory: %s\n"
		} else {
			msg += "cmd directory sibling entries: %+v\n"
		}
		up1CmdDir := filepath.Dir(cmdDir)
		up1Entries := []os.DirEntry{}
		if up1CmdDir != "" {
			up1Entries, err = os.ReadDir(up1CmdDir)
			if err != nil {
				msg += "error reading directory: %s\n"
			} else {
				msg += "cmd directory parent sibling entries: %+v\n"
			}
		}

		o.T.Fatalf(msg, err.Error(), o.Cmd.Path, o.Args, o.Env, o.WorkingDir, entries, up1Entries)
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

func (o *Oneshot) Kill() {
	if o.Cmd != nil {
		o.Cmd.Process.Kill()
		o.Cmd.Process.Wait()
	}
}

func (o *Oneshot) LogToStdErr() {
	o.Env = append(o.Env, "ONESHOT_LOG_STDERR=true")
}

func (o *Oneshot) LogLevel(level string) {
	o.Env = append(o.Env, "ONESHOT_LOG_LEVEL="+level)
}

func (o *Oneshot) StdErrIsTTY() {
	o.Env = append(o.Env, "ONESHOT_TESTING_TTY_STDERR=true")
}

func (o *Oneshot) StdOutIsTTY() {
	o.Env = append(o.Env, "ONESHOT_TESTING_TTY_STDOUT=true")
}

func (o *Oneshot) StdInIsTTY() {
	o.Env = append(o.Env, "ONESHOT_TESTING_TTY_STDIN=true")
}

func (o *Oneshot) IsTTY() {
	o.StdErrIsTTY()
	o.StdOutIsTTY()
}
