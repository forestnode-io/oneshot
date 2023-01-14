package fmt

import (
	"fmt"
	"time"
)

func RoundedDurationString(d time.Duration, digits int) string {
	dd := 1
	for range make([]struct{}, digits) {
		dd *= 10
	}
	ptd := time.Duration(dd)

	switch {
	case d > time.Second:
		d = d.Round(time.Second / ptd)
	case d > time.Millisecond:
		d = d.Round(time.Millisecond / ptd)
	case d > time.Microsecond:
		d = d.Round(time.Microsecond / ptd)
	}
	return d.String()
}

const (
	kb = 1000
	mb = kb * 1000
	gb = mb * 1000
)

func PrettySize(n int64) string {
	var (
		str  string
		size = float64(n)
	)

	// Create the size string using appropriate units: B, KB, MB, and GB
	switch {
	case size < kb:
		str = fmt.Sprintf("%dB", n)
	case size < mb:
		size = size / kb
		str = fmt.Sprintf("%.2fKB", size)
	case size < gb:
		size = size / mb
		str = fmt.Sprintf("%.2fMB", size)
	default:
		size = size / gb
		str = fmt.Sprintf("%.2fGB", size)
	}

	return str
}

// PrettyRate returns a pretty version of n, where n is in bytes per nanosecond
func PrettyRate(n float64) string {
	var (
		str  string
		rate = float64(n)
	)

	// Create the size string using appropriate units: B, KB, MB, and GB
	switch {
	case rate < kb:
		str = fmt.Sprintf("%.2fB/s", rate)
	case rate < mb:
		rate = rate / kb
		str = fmt.Sprintf("%.2fKB/s", rate)
	case rate < gb:
		rate = rate / mb
		str = fmt.Sprintf("%.2fMB/s", rate)
	default:
		rate = rate / gb
		str = fmt.Sprintf("%.2fGB/s", rate)
	}

	return str
}

type Number interface {
	~float32 | ~float64 | ~int | ~int32 | ~int64
}

func PrettyPercent[T Number](x, total T) string {
	if total == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.2f%%", float64(100*x/total))
}

func Address(host, port string) string {
	if port != "" {
		port = ":" + port
	}

	return host + port
}
