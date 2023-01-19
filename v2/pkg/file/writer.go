package file

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
)

type WriteTransferConfig struct {
	userProvidedName string
	dir              string

	w io.WriteCloser
}

func NewWriteTransferConfig(ctx context.Context, location string) (*WriteTransferConfig, error) {
	var wtc WriteTransferConfig
	// if we are receiving to stdout
	if location == "" {
		// and are outputting json
		if format, _ := output.GetFormatAndOpts(ctx); format == "json" {
			// send the contents into the ether.
			// theres a buffer elsewhere that will provide the contents in the json object.
			wtc.w = null{}
		} else {
			// otherwise write the content to stdout
			wtc.w = output.GetBufferedWriteCloser(ctx)
		}
		return &wtc, nil
	}

	var err error
	location, err = filepath.Abs(location)
	if err != nil {
		return nil, err
	}

	// get info on user provided location
	stat, err := os.Stat(location)
	// if location doesnt exist
	if err != nil {
		// then the user wants to receive to a file that doesnt exist yet and has given
		// us the file and directory names all in 1 string
		wtc.dir, wtc.userProvidedName = filepath.Split(location)
	} else {
		// otherwise, the user either wants to receive to an existing directory and
		// hasnt given us the name of the file they want to receive to
		if stat.IsDir() {
			// so we can only set the directory name
			wtc.dir = location
		} else {
			// or the user wants to overwrite an existing file
			wtc.dir, wtc.userProvidedName = filepath.Split(location)
		}
	}

	// make sure we can write to the directory we're receiving to
	if err = isDirWritable(wtc.dir); err != nil {
		return nil, err
	}

	return &wtc, nil
}

type WriteTransferSession struct {
	ctx context.Context

	Progress atomic.Int64
	w        io.WriteCloser
}

func (w *WriteTransferConfig) NewWriteTransferSession(ctx context.Context, name, mime string) (*WriteTransferSession, error) {
	if w.w != nil {
		return &WriteTransferSession{
			ctx: ctx,
			w:   w.w,
		}, nil
	}

	var (
		err error
		ts  = WriteTransferSession{ctx: ctx}
	)

	fileName := w.userProvidedName
	if fileName == "" {
		fileName = name
	}
	if fileName == "" {
		fileName, err = randName(mime)
		if err != nil {
			return nil, err
		}
	}

	path := filepath.Join(w.dir, fileName)
	if ts.w, err = os.Create(path); err != nil {
		return nil, err
	}

	return &ts, nil
}

func (ts *WriteTransferSession) Write(p []byte) (int, error) {
	n, err := ts.w.Write(p)
	ts.Progress.Add(int64(n))
	return n, err
}

func (ts *WriteTransferSession) Close() error {
	err := ts.w.Close()
	if !events.Succeeded(ts.ctx) {
		if file, ok := ts.w.(*os.File); ok && file != nil {
			if file != os.Stdout {
				// TODO(raphaelreyna): handle this
				_ = os.Remove(file.Name())
			}
		}
	}
	return err
}

func (ts *WriteTransferSession) WroteTo() string {
	if !events.Succeeded(ts.ctx) {
		return ""
	}

	if file, ok := ts.w.(*os.File); ok && file != nil {
		if file != os.Stdout {
			return file.Name()
		}
	}

	return ""
}

func randName(mimeType string) (string, error) {
	name := fmt.Sprintf("%0-x", rand.Int31())
	if mimeType != "" {
		// use it get the appropriate file extension
		exts, err := mime.ExtensionsByType(mimeType)
		if err != nil {
			return "", err
		}
		if len(exts) > 0 {
			name += exts[0]
		}
	}
	return name, nil
}

func isDirWritable(path string) error {
	path = filepath.Clean(path)
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
	testFileName := fmt.Sprintf("oneshot%d", time.Now().Unix())
	file, err := os.Create(filepath.Join(path, testFileName))
	if err != nil {
		return err
	}
	file.Close()
	os.Remove(file.Name())
	return nil
}

func _isDirWritable_unix(path string, info os.FileInfo) error {
	const (
		// Owner  Group  Other
		// rwx    rwx    rwx
		bmOthers = 0b000000010 // 000 000 010
		bmGroup  = 0b000010000 // 000 010 000
		bmOwner  = 0b010000000 // 010 000 000
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

	return fmt.Errorf("%s: permission denied %+v - %+v", path, int(mode.Perm()), bmOwner)
}

// null is a noop io.WriteCloser
type null struct{}

func (null) Write(p []byte) (int, error) {
	return len(p), nil
}

func (null) Close() error {
	return nil
}
