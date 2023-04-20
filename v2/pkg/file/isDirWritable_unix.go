//go:build !windows

package file

import (
	"fmt"
	"os"
	"os/user"
	"syscall"
)

func _isDirWritable(path string, info os.FileInfo) error {
	const (
		// Owner  Group  Other
		// rwx    rwx    rwx
		bmOthers = 0b000000010 // 000 000 010
		bmGroup  = 0b000010000 // 000 010 000
		bmOwner  = 0b010000000 // 010 000 000
	)
	var mode = info.Mode()

	// check if writable by others
	if mode&bmOthers != 0 {
		return nil
	}

	stat := info.Sys().(*syscall.Stat_t)
	usr, err := user.Current()
	if err != nil {
		return err
	}

	// check if writable by group
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

	// check if writable by owner
	if mode&bmOwner != 0 {
		uid := fmt.Sprint(stat.Uid)
		if uid == usr.Uid {
			return nil
		}
	}

	return fmt.Errorf("%s: permission denied %+v - %+v", path, int(mode.Perm()), bmOwner)
}
