package output

import (
	"context"
	"fmt"
	"time"
)

func DisplaySpinner(ctx context.Context, period time.Duration, prefix, succ string, charSet []string) func() {
	o := getOutput(ctx)
	if o.quiet || o.Format == "json" {
		return func() {}
	}

	var (
		out   = o.dynamicOutput
		done  chan struct{}
		csLen = len(charSet)
	)

	if out != nil {
		done = make(chan struct{})
		ticker := time.NewTicker(period)

		go func() {
			out.resetLine()
			fmt.Printf("%s %s", prefix, charSet[0])
			out.flush()

			idx := 1
			for {
				select {
				case <-done:
					ticker.Stop()
					return
				case <-ticker.C:
					dyn := charSet[idx%csLen]
					idx++

					out.resetLine()
					fmt.Fprintf(out, "%s %s", prefix, dyn)
					out.flush()
				}
			}
		}()
	}

	return func() {
		if done != nil {
			done <- struct{}{}
			close(done)
			done = nil
		}

		out.resetLine()
		fmt.Fprintln(out, succ)
		out.flush()
	}
}
