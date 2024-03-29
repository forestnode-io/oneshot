package output

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	oneshotfmt "github.com/forestnode-io/oneshot/v2/pkg/output/fmt"
)

const progDisplayTimeFormat = "2006-01-02T15:04:05-0700"

func displayDynamicProgress(o *output, prefix string, start time.Time, prog *atomic.Int64, total int64) time.Time {
	var (
		progress = prog.Load()
		out      = o.dynamicOutput
		rate     float64
	)

	rate = bytesPerSecond(progress-o.lastProgressDisplayAmount, o.displayProgresssPeriod)
	o.lastProgressDisplayAmount = progress

	o.dynamicOutput.resetLine()
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

	out.flush()

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

func _displayFlush(o *output, s string, success bool) {
	// if we were dynamically displaying progress to stderr
	if o.dynamicOutput != nil {
		// update the progress there
		o.dynamicOutput.resetLine()
		if color := o.stderrFailColor; !success && color != nil {
			payload := o.dynamicOutput.String(s)
			payload = payload.Foreground(color)
			fmt.Fprint(o.dynamicOutput, payload)
		} else {
			fmt.Fprint(o.dynamicOutput, s)
		}

		o.dynamicOutput.flush()
	} else {
		// otherwise, just print to stderr
		if color := o.stderrFailColor; !success && color != nil && o.stderrTTY != nil {
			payload := o.stderrTTY.String(s)
			payload = payload.Foreground(color)
			fmt.Fprint(o.stderrTTY, payload)
		} else {
			fmt.Fprint(os.Stderr, s)
		}
	}
}
