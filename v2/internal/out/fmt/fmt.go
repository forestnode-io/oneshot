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
		str = fmt.Sprintf("%d B", n)
	case size < mb:
		size = size / kb
		str = fmt.Sprintf("%.3f KB", size)
	case size < gb:
		size = size / mb
		str = fmt.Sprintf("%.3f MB", size)
	default:
		size = size / gb
		str = fmt.Sprintf("%.3f GB", size)
	}

	return str
}

func PrettyRate(n int64) string {
	var (
		str  string
		rate = float64(n)
	)

	// Create the size string using appropriate units: B, KB, MB, and GB
	switch {
	case rate < kb:
		str = fmt.Sprintf("%.2f B/s", rate)
	case rate < mb:
		rate = rate / kb
		str = fmt.Sprintf("%.2f KB/s", rate)
	case rate < gb:
		rate = rate / mb
		str = fmt.Sprintf("%.2f MB/s", rate)
	default:
		rate = rate / gb
		str = fmt.Sprintf("%.2f GB/s", rate)
	}

	return str
}

func Address(host, port string) string {
	if port != "" {
		port = ":" + port
	}

	return host + port
}
