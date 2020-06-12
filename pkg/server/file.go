package server

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime"
	"os"
	"path/filepath"
)

// File represents the file being transferred, whether its from an
// actual file or stdin. File also holds the files metadata.
type File struct {
	// Path is optional if Name, Ext and MimeType are provided
	Path string

	// Name, Ext and MimeType are optional if Path is provided
	Name     string
	Ext      string
	MimeType string

	file   *os.File
	buffer *bytes.Reader

	size           int64
	progress       int64
	ProgressWriter io.Writer
}

func (f *File) Open() error {
	if f.file != nil {
		return nil
	}

	switch f.Path {
	case "":
		f.file = os.Stdin
		if f.Name == "" {
			f.Name = fmt.Sprintf("%0-x", rand.Int31())
		}
		dataBytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		f.buffer = bytes.NewReader(dataBytes)
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

	return nil
}

func (f *File) Close() error {
	if f.file == nil {
		return nil
	}
	return f.file.Close()
}

func (f *File) Size() int64 {
	return f.size
}

func (f *File) Read(p []byte) (n int, err error) {
	err = f.Open()
	if err != nil {
		return
	}

	if f.progress == 0 {
		f.ProgressWriter.Write([]byte("\n"))
		f.WriteProgress()
	}

	if f.buffer != nil {
		n, err = f.buffer.Read(p)
	} else {
		n, err = f.file.Read(p)
	}

	f.progress += int64(n)
	f.WriteProgress()

	if err == io.EOF && f.ProgressWriter != nil {
		fmt.Fprint(f.ProgressWriter, "\n")
	}

	return
}

func (f *File) WriteProgress() {
	if f.ProgressWriter == nil {
		return
	}
	fmt.Fprintf(f.ProgressWriter, "download progress: %.2f%%\r",
		100.0*float64(f.progress)/float64(f.size),
	)
}
