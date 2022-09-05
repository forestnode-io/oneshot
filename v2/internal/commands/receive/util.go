package receive

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"syscall"
	"time"
)

var (
	lf   = []byte{10}
	crlf = []byte{13, 10}
)

var regex = regexp.MustCompile(`filename="(.+)"`)

func fileName(s string) string {
	subs := regex.FindStringSubmatch(s)
	if len(subs) > 1 {
		return subs[1]
	}
	return ""
}

func isDirWritable(path string) error {
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	if runtime.GOOS == "windows" {
		return _isDirWritable_windows(path, info)
	}
	return _isDirWritable_unix(path, info)
}

func _isDirWritable_windows(path string, info os.FileInfo) error {
	testFileName := fmt.Sprintf("oneshot%d", time.Now().Unix())
	file, err := os.Create(testFileName)
	if err != nil {
		return err
	}
	file.Close()
	os.Remove(file.Name())
	return nil
}

func _isDirWritable_unix(path string, info os.FileInfo) error {
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
