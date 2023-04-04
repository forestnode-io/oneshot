package sys

import "runtime"

// copied from https://github.com/golang/go/blob/ebb572d82f97d19d0016a49956eb1fddc658eb76/src/go/build/syslist.go#L38
var unixOS = map[string]struct{}{
	"aix":       {},
	"android":   {},
	"darwin":    {},
	"dragonfly": {},
	"freebsd":   {},
	"hurd":      {},
	"illumos":   {},
	"ios":       {},
	"linux":     {},
	"netbsd":    {},
	"openbsd":   {},
	"solaris":   {},
}

func RunningOnUNIX() bool {
	_, isUnix := unixOS[runtime.GOOS]
	return isUnix
}
