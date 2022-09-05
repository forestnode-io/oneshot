package summary

import (
	"fmt"
)

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
		str = fmt.Sprintf("%.3f B/s", rate)
	case rate < mb:
		rate = rate / kb
		str = fmt.Sprintf("%.3f KB/s", rate)
	case rate < gb:
		rate = rate / mb
		str = fmt.Sprintf("%.3f MB/s", rate)
	default:
		rate = rate / gb
		str = fmt.Sprintf("%.3f GB/s", rate)
	}

	return str
}
