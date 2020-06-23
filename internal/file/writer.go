package file

import (
	"fmt"
	"io"
	"math/rand"
	"mime"
	"os"
	"path/filepath"
	"sync"
)

// FileWriter represents the file being received, whether its to an
// actual file or stdout. File also holds the files metadata.
type FileWriter struct {
	// Path is optional if Name, Ext and MimeType are provided
	Path string
	// Name is the filename to use when writing to disk
	name string

	MIMEType string

	// ProgressWriter will be used to output read progress
	// whenever this File structs Read() method is called.
	ProgressWriter io.Writer

	location string // path file on disk

	userProvidedName bool

	file     *os.File
	size     int64
	progress int64
	sync.Mutex
}

func (f *FileWriter) Close() error {
	if f.file == nil {
		return nil
	}
	return f.file.Close()
}

func (f *FileWriter) GetSize() int64 {
	if f.size == 0 {
		return f.progress
	}
	return f.size
}

func (f *FileWriter) SetSize(size int64) {
	f.size = size
}

func (f *FileWriter) GetLocation() string {
	return f.location
}

func (f *FileWriter) Name() string {
	return f.name
}

func (f *FileWriter) SetName(name string, fromRemote bool) {
	f.name = name
	if !fromRemote {
		f.userProvidedName = true
	}
}

// Open prepares the files contents for reading.
// If f.file is the empty string then f.Open() will read from stdin into a buffer.
// This method is idempotent.
func (f *FileWriter) Open() error {
	if f.file != nil {
		return nil
	}

	switch f.Path {
	case "":
		f.file = os.Stdout
		return nil
	default:
		if f.name == "" {
			f.name = fmt.Sprintf("%0-x", rand.Int31())
			if f.MIMEType != "" {
				exts, err := mime.ExtensionsByType(f.MIMEType)
				if err != nil {
					return err
				}
				if len(exts) > 0 {
					f.name = f.name + exts[0]
				}
			}
		}
		f.location = filepath.Join(f.Path, f.name)
	}

	var err error
	if f.file, err = os.Create(f.location); err != nil {
		return err
	}

	return nil
}

func (f *FileWriter) Write(p []byte) (n int, err error) {
	if f.file == nil {
		return 0, UnopenedReadErr
	}

	if f.progress == 0 {
		f.newline()
		f.writeProgress()
	}

	n, err = f.file.Write(p)

	f.progress += int64(n)
	f.writeProgress()

	return
}

func (f *FileWriter) Reset() error {
	if f.file == nil {
		return nil
	}

	f.Close()
	f.file = nil
	if !f.userProvidedName {
		f.name = ""
	}
	f.progress = 0
	if f.location != "" {
		os.Remove(f.location)
	}
	f.location = ""
	return nil
}

func (f *FileWriter) newline() {
	if f.ProgressWriter != nil {
		f.ProgressWriter.Write([]byte("\n"))
	}
}

func (f *FileWriter) writeProgress() {
	const (
		kb = 1000
		mb = kb * 1000
		gb = mb * 1000
	)
	if f.ProgressWriter == nil || f.Path == "" {
		return
	}
	switch {
	case f.size == 0:
		switch {
		case f.progress < kb:
			fmt.Fprintf(f.ProgressWriter, "transferred: %d B\r", f.progress)
		case f.progress < mb:
			fmt.Fprintf(f.ProgressWriter, "transferred: %.3f KB\r", float64(f.progress)/kb)
		case f.progress < gb:
			fmt.Fprintf(f.ProgressWriter, "transferred: %.3f MB\r", float64(f.progress)/mb)
		default:
			fmt.Fprintf(f.ProgressWriter, "transferred: %.3f GB\r", float64(f.progress)/gb)
		}
		return
	default:
		fmt.Fprintf(f.ProgressWriter, "transfer progress: %.2f%%\r",
			100.0*float64(f.progress)/float64(f.size),
		)
	}
}
