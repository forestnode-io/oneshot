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

	file *os.File
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
	default:
		var err error
		f.file, err = os.Open(f.Path)
		if err != nil {
			return err
		}
		if f.Name == "" {
			f.Name = filepath.Base(f.Path)
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

func (f *File) Read(p []byte) (n int, err error) {
	err = f.Open()
	if err != nil {
		return 0, err
	}
	return f.file.Read(p)
}
