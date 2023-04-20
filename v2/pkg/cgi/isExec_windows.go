//go:build windows

package cgi

import (
	"fmt"
	"path/filepath"
)

func isExec(path string) error {
	if filepath.Ext(path) == ".exe" {
		return nil
	}

	return fmt.Errorf("%s must be executable", path)
}
