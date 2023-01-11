package output

import (
	"fmt"
	"sync/atomic"
	"time"

	oneshotfmt "github.com/raphaelreyna/oneshot/v2/pkg/output/fmt"
)

const progDisplayTimeFormat = "2006-01-02T15:04:05-0700"

func displayProgressSuccessFlush(o *output, prefix string, start time.Time, total int64) {
	const (
		kb = 1000
		mb = kb * 1000
		gb = mb * 1000
	)

	if o.stderrIsTTY {
		o.Stderr.ClearLineRight()
		o.Stderr.RestoreCursorPosition()
	} else if o.stdoutIsTTY {
		o.Stderr.ClearLineRight()
		o.Stderr.RestoreCursorPosition()
	}

	if !o.stdoutIsTTY && o.stderrIsTTY {
		fmt.Fprint(o.Stderr, prefix)
		switch {
		case total < kb:
			fmt.Fprintf(o.Stderr, "%8d B  ", total)
		case total < mb:
			fmt.Fprintf(o.Stderr, "%8.2f KB  ", float64(total)/kb)
		case total < gb:
			fmt.Fprintf(o.Stderr, "%8.2f MB  ", float64(total)/mb)
		default:
			fmt.Fprintf(o.Stderr, "%8.2f GB  ", float64(total)/gb)
		}

		duration := time.Since(start)
		rate := 1000 * 1000 * 1000 * total / int64(duration)
		fmt.Fprintf(o.Stderr, "%v  100%%  0s  %v  ...success\n",
			oneshotfmt.PrettyRate(rate),
			oneshotfmt.RoundedDurationString(duration, 2),
		)
	}
	fmt.Fprint(o.Stdout, prefix)

	switch {
	case total < kb:
		fmt.Fprintf(o.Stdout, "%8d B  ", total)
	case total < mb:
		fmt.Fprintf(o.Stdout, "%8.2f KB  ", float64(total)/kb)
	case total < gb:
		fmt.Fprintf(o.Stdout, "%8.2f MB  ", float64(total)/mb)
	default:
		fmt.Fprintf(o.Stdout, "%8.2f GB  ", float64(total)/gb)
	}

	duration := time.Since(start)
	rate := 1000 * 1000 * 1000 * total / int64(duration)
	fmt.Fprintf(o.Stdout, "%v  100%%  0s  %v  ...success\n",
		oneshotfmt.PrettyRate(rate),
		oneshotfmt.RoundedDurationString(duration, 2),
	)
}

func displayProgress(o *output, prefix string, start time.Time, prog *atomic.Int64, total int64) time.Time {
	const (
		kb = 1000
		mb = kb * 1000
		gb = mb * 1000
	)

	var progress = prog.Load()

	o.Stderr.ClearLineRight()
	o.Stderr.RestoreCursorPosition()

	fmt.Fprint(o.Stderr, prefix)

	switch {
	case progress < kb:
		fmt.Fprintf(o.Stderr, "%8d B  ", progress)
	case progress < mb:
		fmt.Fprintf(o.Stderr, "%8.2f KB  ", float64(progress)/kb)
	case progress < gb:
		fmt.Fprintf(o.Stderr, "%8.2f MB  ", float64(progress)/mb)
	default:
		fmt.Fprintf(o.Stderr, "%8.2f GB  ", float64(progress)/gb)
	}

	duration := time.Since(start)
	rate := 1000 * 1000 * 1000 * progress / int64(duration)
	fmt.Fprintf(o.Stderr, "%v  ", oneshotfmt.PrettyRate(rate))
	if total != 0 {
		percent := 100.0 * float64(progress) / float64(total)
		if rate != 0 {
			timeLeft := (total - progress) / rate
			fmt.Fprintf(o.Stderr, "%8.2f%%  %d  ", percent, timeLeft)
		} else {
			fmt.Fprintf(o.Stderr, "%8.2f%%  n/a  ", percent)
		}
	} else {
		fmt.Fprintf(o.Stderr, "n/a  n/a  ")
	}
	fmt.Fprintf(o.Stderr, "%v  ", oneshotfmt.RoundedDurationString(duration, 2))

	return start
}
