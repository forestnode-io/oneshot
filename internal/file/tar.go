package file

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func tarball(path string, w io.Writer) error {
	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.Walk(path, func(fp string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}

		header := tar.Header{
			Name:    strings.TrimPrefix(fp, filepath.Dir(path)),
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}
		if err := tw.WriteHeader(&header); err != nil {
			return err
		}

		currFile, err := os.Open(fp)
		if err != nil {
			return err
		}

		_, err = io.Copy(tw, currFile)
		currFile.Close()
		if err != nil {
			return err
		}

		return nil
	})
}
