package server

import (
	"fmt"
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

	file      *os.File
	size      int64
	measuring bool
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
		f.measuring = true
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
	n, err = f.file.Read(p)
	if f.measuring {
		f.size += int64(n)
	}

	return
}
