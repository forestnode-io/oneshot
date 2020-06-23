package file

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime"
	"os"
	"path/filepath"
	"sync"
)

var UnopenedReadErr = errors.New("attempted to read unopened file")

// FileReader represents the file being sent, whether its from an
// actual file or stdin. FileReader also holds the files metadata.
type FileReader struct {
	// Path is optional if Name, Ext and MimeType are provided
	Path string

	// Name is optional if Path is provided
	Name string
	// Ext is optional if Path is provided
	Ext string
	// MimeType is optional if Path is provided
	MimeType string

	// ProgressWriter will be used to output read progress
	// whenever this File structs Read() method is called.
	ProgressWriter io.Writer

	file        *os.File
	buffer      *bytes.Reader
	bufferBytes []byte
	size        int64
	progress    int64
	mutex       *sync.Mutex

	requestCount int
	readCount    int
}

func (f *FileReader) Lock() {
	if f.mutex == nil {
		f.mutex = &sync.Mutex{}
	}
	f.mutex.Lock()
}

func (f *FileReader) Unlock() {
	if f.mutex == nil {
		f.mutex = &sync.Mutex{}
	}
	f.mutex.Unlock()
}

func (f *FileReader) Close() error {
	if f.file == nil {
		return nil
	}
	return f.file.Close()
}

func (f *FileReader) Size() int64 {
	return f.size
}

func (f *FileReader) RequestCount() int {
	return f.requestCount
}

// Requested increases the request count by one
func (f *FileReader) Requested() {
	f.requestCount++
}

// ReadCount returns how many times the file has been read
func (f *FileReader) ReadCount() int {
	return f.readCount
}

// Open prepares the files contents for reading.
// If f.file is the empty string then f.Open() will read from stdin into a buffer.
// This method is idempotent.
func (f *FileReader) Open() error {
	var err error
	if f.file != nil {
		return nil
	}

	switch f.Path {
	case "":
		f.file = os.Stdin
		if f.Name == "" {
			f.Name = fmt.Sprintf("%0-x", rand.Int31())
		}
		f.bufferBytes, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		f.buffer = bytes.NewReader(f.bufferBytes)
		f.size = f.buffer.Size()
	default:
		var err error
		f.file, err = os.Open(f.Path)
		if err != nil {
			return err
		}
		info, err := f.file.Stat()
		if err != nil {
			return err
		}
		f.size = info.Size()
		if f.Name == "" {
			f.Name = info.Name()
		}
	}

	if f.Ext == "" {
		f.Ext = filepath.Ext(f.Name)
	} else if f.Ext[0] != '.' {
		f.Ext = "." + f.Ext
	}

	if f.MimeType == "" {
		f.MimeType = mime.TypeByExtension(f.Ext)
	}
	if f.MimeType == "" {
		f.MimeType = "text/plain"
	}

	return nil
}

func (f *FileReader) Read(p []byte) (n int, err error) {
	if f.file == nil {
		return 0, UnopenedReadErr
	}

	if f.progress == 0 {
		f.ProgressWriter.Write([]byte("\n"))
		f.writeProgress()
	}

	if f.buffer != nil {
		n, err = f.buffer.Read(p)
	} else {
		n, err = f.file.Read(p)
	}

	f.progress += int64(n)
	f.writeProgress()

	if err == io.EOF && f.ProgressWriter != nil {
		f.readCount++
		fmt.Fprint(f.ProgressWriter, "\n")
	}

	return
}

func (f *FileReader) Reset() error {
	if f.file == nil {
		return nil
	}

	err := f.Close()
	if err != nil {
		return err
	}
	f.file = nil
	f.progress = 0
	return err
}

func (f *FileReader) newline() {
	if f.ProgressWriter != nil {
		f.ProgressWriter.Write([]byte("\n"))
	}
}

func (f *FileReader) writeProgress() {
	if f.ProgressWriter == nil {
		return
	}
	fmt.Fprintf(f.ProgressWriter, "transfer progress: %.2f%%\r",
		100.0*float64(f.progress)/float64(f.size),
	)
}
