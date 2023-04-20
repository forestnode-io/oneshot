//go:build !windows

package cgi

import (
	"fmt"
	"os"
	"os/user"
	"syscall"
)

func isExec(path string) error {
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
