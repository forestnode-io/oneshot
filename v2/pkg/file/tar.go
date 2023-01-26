package file

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func tarball(compress bool, paths []string, w io.Writer) error {
	var tw *tar.Writer
	if compress {
		gw := gzip.NewWriter(w)
		defer gw.Close()
		tw = tar.NewWriter(gw)
	} else {
		tw = tar.NewWriter(w)
	}
	defer tw.Close()

	formatName := func(name string) string {
		// needed for windows
		name = strings.ReplaceAll(name, `\`, `/`)
		if string(name[0]) == `/` {
			name = name[1:]
		}
		return name
	}

	writeFile := func(path, name string, info os.FileInfo) error {
		header := tar.Header{
			Name:    name,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}
		if err := tw.WriteHeader(&header); err != nil {
			return err
		}

		currFile, err := os.Open(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(tw, currFile)
		currFile.Close()
		if err != nil {
			return err
		}

		return nil
	}

	walkFunc := func(path string) func(string, os.FileInfo, error) error {
		dir := filepath.Dir(path)
		return func(fp string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			} else if err != nil {
				return err
			}

			name := strings.TrimPrefix(fp, dir)
			name = formatName(name)

			if err = writeFile(fp, name, info); err != nil {
				return err
			}

			return nil
		}
	}

	// Loop over files to be archived
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() { // Archiving a file
			name := filepath.Base(path)
			name = formatName(name)

			err = writeFile(path, name, info)
			if err != nil {
				return err
			}
		} else { // Archiving a directory; needs to be walked
			err := filepath.Walk(path, walkFunc(path))
			if err != nil {
				return err
			}
		}
	}

	return nil
}
