//go:build windows

package file

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func _isDirWritable(path string, info os.FileInfo) error {
	testFileName := fmt.Sprintf("oneshot%d", time.Now().Unix())
	file, err := os.Create(filepath.Join(path, testFileName))
	if err != nil {
		return err
	}
	file.Close()
	os.Remove(file.Name())
	return nil
}
