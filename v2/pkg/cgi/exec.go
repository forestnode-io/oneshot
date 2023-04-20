package cgi

import (
	"os"
	"path/filepath"
	"strings"
)

func findExec(name string) string {
	if name != filepath.Base(name) {
		if err := isExec(name); err == nil {
			return name
		}
	} else {
		paths := strings.Split(os.Getenv("PATH"), ":")
		for _, pathDir := range paths {
			execPath := filepath.Join(pathDir, name)
			if err := isExec(execPath); err == nil {
				return execPath
			}
		}
	}

	return ""
}
