package output

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/muesli/termenv"
	oneshotfmt "github.com/raphaelreyna/oneshot/v2/pkg/output/fmt"
)

const progDisplayTimeFormat = "2006-01-02T15:04:05-0700"
const (
	kb = 1000
	mb = kb * 1000
	gb = mb * 1000
)

func displayProgress(o *output, prefix string, start time.Time, prog *atomic.Int64, total int64) time.Time {
	var (
		progress = prog.Load()
		out      = o.tabbedStderr
		rate     float64
	)

	rate = bytesPerSecond(progress-o.lastProgressDisplayAmount, o.displayProgresssPeriod)
	o.lastProgressDisplayAmount = progress

	_displayResetLine(o.Stderr)
	fmt.Fprint(out, prefix)

	var (
		duration       = time.Since(start)
		durationString = oneshotfmt.RoundedDurationString(duration, 2)
		sizeString     = oneshotfmt.PrettySize(progress)
		rateString     = oneshotfmt.PrettyRate(rate)
	)

	if total != 0 {
		percent := oneshotfmt.PrettyPercent(progress, total)
		if 1 <= rate {
			deltaBytes := total - progress
			timeLeft := deltaBytes / int64(rate) // [B] / ( [B/s] ) = [s]
			fmt.Fprintf(out, "\t%v\t%v\t%s\t%s\t%s", sizeString, rateString, percent, durationString, time.Duration(timeLeft)*time.Second)
		} else {
			fmt.Fprintf(out, "\t%v\t%v\t%s\t%s\tn/a", sizeString, rateString, oneshotfmt.PrettyPercent(progress, total), durationString)
		}
	} else {
		fmt.Fprintf(out, "\t%v\t%v\tn/a\t%s\tn/a", sizeString, rateString, durationString)
	}

	out.Flush()

	return start
}

func displayProgressSuccessFlush(o *output, prefix string, start time.Time, total int64) {
	duration := time.Since(start)
	tail := fmt.Sprintf("\t%s\t%v\t100%%\t%v\tsuccess\n",
		oneshotfmt.PrettySize(total),
		oneshotfmt.PrettyRate(bytesPerSecond(total, duration)),
		oneshotfmt.RoundedDurationString(duration, 2),
	)
	_displayFlush(o, prefix+tail, true)
}

func displayProgressFailFlush(o *output, prefix string, start time.Time, prog, total int64) {
	duration := time.Since(start)
	tail := fmt.Sprintf("\t%s\t%v\t%s\t%v\tfail\n",
		oneshotfmt.PrettySize(prog),
		oneshotfmt.PrettyRate(bytesPerSecond(prog, duration)),
		oneshotfmt.PrettyPercent(prog, total),
		oneshotfmt.RoundedDurationString(duration, 2),
	)
	_displayFlush(o, prefix+tail, false)
}

func _displayResetLine(o *termenv.Output) {
	o.WriteString("\r")
	o.ClearLineRight()
}

func _displayFlush(o *output, s string, success bool) {
	// if we were dynamically displaying progress
	if o.stderrIsTTY {
		// and stderr is the only tty
		if !o.stdoutIsTTY {
			// update the progress via stderr
			_displayResetLine(o.Stderr)
			if color := o.stderrFailColor; !success && color != nil {
				payload := o.Stderr.String(s)
				payload = payload.Foreground(color)
				fmt.Fprint(o.tabbedStderr, payload)
			} else {
				fmt.Fprint(o.tabbedStderr, s)
			}
			o.tabbedStderr.Flush()
		}
	}

	// print to stdout
	if o.stdoutIsTTY {
		_displayResetLine(o.Stdout)
		if color := o.stdoutFailColor; !success && color != nil {
			s = o.Stdout.String(s).Foreground(color).String()
		}
	}

	fmt.Fprint(o.tabbedStdout, s)
	o.tabbedStdout.Flush()
}
