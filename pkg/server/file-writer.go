package server

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

	// Name is optional if Path points to a file
	Name string
	// Ext is optional if Path is provided
	Ext      string
	MimeType string

	// ProgressWriter will be used to output read progress
	// whenever this File structs Read() method is called.
	ProgressWriter io.Writer

	file     *os.File
	Size     int64
	progress int64
	mutex    *sync.Mutex
}

func (f *FileWriter) Lock() {
	if f.mutex == nil {
		f.mutex = &sync.Mutex{}
	}
	f.mutex.Lock()
}

func (f *FileWriter) Unlock() {
	if f.mutex == nil {
		f.mutex = &sync.Mutex{}
	}
	f.mutex.Unlock()
}

func (f *FileWriter) Close() error {
	if f.file == nil {
		return nil
	}
	return f.file.Close()
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
	default:
		stat, err := os.Stat(f.Path)
		if err != nil {
			return err
		}

		if stat.IsDir() {
			if f.Name == "" {
				f.Name = fmt.Sprintf("%0-x", rand.Int31())
			}
			if f.MimeType != "" && f.Ext == "" {
				exts, err := mime.ExtensionsByType(f.MimeType)
				if err != nil {
					return err
				}
				f.Ext = exts[0]
				f.Name = f.Name + f.Ext
			} else if f.Ext != "" {
				f.Name = f.Name + f.Ext
			}
			f.Path = filepath.Join(f.Path, f.Name)
		}

		f.file, err = os.Create(f.Path)
		if err != nil {
			return err
		}
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

	n, err = f.Write(p)

	f.progress += int64(n)
	f.writeProgress()

	return
}

func (f *FileWriter) Reset() error {
	if f.file == nil {
		return nil
	}

	err := f.Close()
	if err != nil {
		return err
	}
	f.file = nil
	f.progress = 0
	if f.Path != "" {
		os.Remove(f.Path)
	}
	return err
}

func (f *FileWriter) newline() {
	if f.ProgressWriter != nil {
		f.ProgressWriter.Write([]byte("\n"))
	}
}

func (f *FileWriter) writeProgress() {
	if f.ProgressWriter == nil {
		return
	}
	fmt.Fprintf(f.ProgressWriter, "transfer progress: %.2f%%\r",
		100.0*float64(f.progress)/float64(f.Size),
	)
}
