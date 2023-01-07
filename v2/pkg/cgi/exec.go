package cgi

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
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

func isExec(path string) error {
	if runtime.GOOS == "windows" {
		return _isExec_windows(path)
	}
	return _isExec_unix(path)
}

func _isExec_unix(path string) error {
	const (
		bmOthers = 0x0001
		bmGroup  = 0x0010
		bmOwner  = 0x0100
	)

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := info.Mode()

	// check if executable by others
	if mode&bmOthers != 0 {
		return nil
	}

	stat := info.Sys().(*syscall.Stat_t)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	// check if executable by group
	if mode&bmGroup != 0 {
		gid := fmt.Sprint(stat.Gid)
		gids, err := usr.GroupIds()
		if err != nil {
			return err
		}
		for _, g := range gids {
			if g == gid {
				return nil
			}
		}
	}

	// check if exec by owner
	if mode&bmOwner != 0 {
		uid := fmt.Sprint(stat.Uid)
		if uid == usr.Uid {
			return nil
		}
	}

	return fmt.Errorf("%s: permission denied", path)
}

func _isExec_windows(path string) error {
	if filepath.Ext(path) == ".exe" {
		return nil
	}

	return fmt.Errorf("%s must be executable", path)
}
