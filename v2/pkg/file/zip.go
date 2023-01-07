package file

import (
	z "archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func zip(paths []string, w io.Writer) error {
	zw := z.NewWriter(w)
	defer zw.Close()

	formatName := func(name string) string {
		// needed for windows
		name = strings.ReplaceAll(name, `\`, `/`)
		if string(name[0]) == `/` {
			name = name[1:]
		}
		return name
	}

	writeFile := func(path, name string, info os.FileInfo) error {
		zFile, err := zw.Create(name)
		if err != nil {
			return err
		}

		currFile, err := os.Open(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(zFile, currFile)
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
