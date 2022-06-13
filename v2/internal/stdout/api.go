package stdout

import (
	"context"
	"strings"

	"github.com/raphaelreyna/oneshot/v2/internal/summary"
	"github.com/spf13/cobra"
)

type stdoutKey struct{}

func WithStdout(ctx context.Context) context.Context {
	return context.WithValue(ctx, stdoutKey{}, &stdout{})
}

func ReceivingToStdout(ctx context.Context) {
	if stdout := stdoutFromContext(ctx); stdout != nil {
		stdout.receivingToStdout = true
	}
}

func WantsJSON(ctx context.Context, opts string) {
	if stdout := stdoutFromContext(ctx); stdout != nil {
		stdout.wantsJSON = true
	}
}

func WriteListeningOn(ctx context.Context, scheme, host, port string) {
	if stdout := stdoutFromContext(ctx); stdout != nil {
		stdout.writeListeningOn(scheme, host, port)
	}
}

func SetCobraCommandStdout(cmd *cobra.Command) {
	var ctx = cmd.Context()
	if stdout := stdoutFromContext(ctx); stdout != nil {
		stdout.w = cmd.OutOrStdout()
		cmd.SetOut(stdout)
	}
}

func WriteSummary(ctx context.Context, summary *summary.Summary) {
	stdout := stdoutFromContext(ctx)
	if stdout == nil {
		return
	}

	if stdout.receivingToStdout {
		return
	}

	if stdout.wantsJSON {
		summary.WriteJSON(stdout.w, strings.Contains(stdout.jopts, "pretty"))
	} else {
		summary.WriteHuman(stdout.w)
	}
}

func stdoutFromContext(ctx context.Context) *stdout {
	stdout, _ := ctx.Value(stdoutKey{}).(*stdout)
	return stdout
}
