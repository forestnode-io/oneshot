package file

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/raphaelreyna/oneshot/v2/internal/out"
)

// FileWriter represents the file being received, whether its to an
// actual file or stdout. File also holds the files metadata.
type FileWriter struct {
	// Path is optional if Name, Ext and MimeType are provided
	Path string
	// Name is the filename to use when writing to disk
	clientProvidedName string
	userProvidedName   string

	MIMEType string

	Progress atomic.Int64

	location string // path file on disk

	file io.WriteCloser
	size int64
	sync.Mutex
}

func (f *FileWriter) Close() error {
	if f.file == nil {
		return nil
	}
	return f.file.Close()
}

func (f *FileWriter) GetSize() int64 {
	return f.size
}

func (f *FileWriter) SetSize(size int64) {
	f.size = size
}

func (f *FileWriter) GetLocation() string {
	return f.location
}

// Name returns the name of the file, giving presedence to the client provided name
func (f *FileWriter) Name() string {
	if f.userProvidedName != "" {
		return f.userProvidedName
	}
	return f.clientProvidedName
}

func (f *FileWriter) ClientProvidedName() string {
	return f.clientProvidedName
}

func (f *FileWriter) UserProvidedName() string {
	return f.clientProvidedName
}

func (f *FileWriter) SetClientProvidedName(name string) {
	f.clientProvidedName = name
}

func (f *FileWriter) SetUserProvidedName(name string) {
	f.userProvidedName = name
}

// Open prepares the files contents for reading.
// If f.file is the empty string then f.Open() will read from stdin into a buffer.
// This method is idempotent.
func (f *FileWriter) Open(ctx context.Context) error {
	if f.file != nil {
		return nil
	}

	// if we are receiving to stdout
	if out.IsServingToStdout(ctx) {
		// and are outputting json
		if format, _ := out.GetFormatAndOpts(ctx); format == "json" {
			// send the contents into the ether.
			// theres a buffer elsewhere that will provide the contents in the json object.
			f.file = null{}
		} else {
			// otherwise write the content to stdout
			f.file = out.GetWriteCloser(ctx)
		}
		return nil
	}

	name := f.Name()
	// if the file wasnt given a name
	if name == "" {
		// create a random one
		name = fmt.Sprintf("%0-x", rand.Int31())

		// if the mime type was provided
		if f.MIMEType != "" {
			// use it get the appropriate file extension
			exts, err := mime.ExtensionsByType(f.MIMEType)
			if err != nil {
				return err
			}
			if len(exts) > 0 {
				name += exts[0]
			}
		}
	}
	f.location = filepath.Join(f.Path, name)

	var err error
	if f.file, err = os.Create(f.location); err != nil {
		return err
	}

	return nil
}

func (f *FileWriter) Write(p []byte) (n int, err error) {
	if f.file == nil {
		return 0, ErrUnopenedRead
	}

	n, err = f.file.Write(p)
	f.Progress.Add(int64(n))

	return
}

func (f *FileWriter) Reset() error {
	if f.file == nil {
		return nil
	}

	f.Close()
	f.file = nil
	f.clientProvidedName = ""
	f.Progress.Store(0)
	if f.location != "" {
		os.Remove(f.location)
	}
	f.location = ""
	return nil
}

// null is a noop io.WriteCloser
type null struct{}

func (null) Write(p []byte) (int, error) {
	return len(p), nil
}

func (null) Close() error {
	return nil
}
