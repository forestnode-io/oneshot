package version

import "os"

var (
	Version    string
	APIVersion string
	Credit     string
	License    = "Apache License 2.0"
)

func init() {
	if os.Getenv("ONESHOT_SKIP_INIT_CHECKS") != "" {
		return
	}

	if Version == "" {
		panic("Version not set")
	}
	if APIVersion == "" {
		panic("APIVersion not set")
	}
	if License == "" {
		panic("License not set")
	}
}
