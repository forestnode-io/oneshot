package stdout

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"
)

type stdoutKey struct{}

func WithStdout(ctx context.Context) context.Context {
	return context.WithValue(ctx, stdoutKey{}, &stdout{})
}

func ReceivingToStdout(ctx context.Context) {
	if stdout := stdoutFromContext(ctx); stdout != nil {
		stdout.skipSummary = true
		if stdout.wantsJSON {
			if stdout.receivedBuf == nil {
				stdout.receivedBuf = bytes.NewBuffer(nil)
			}
		}
	}
}

func WantsJSON(ctx context.Context, opts ...string) {
	if stdout := stdoutFromContext(ctx); stdout != nil {
		stdout.wantsJSON = true
		if stdout.skipSummary {
			if stdout.receivedBuf == nil {
				stdout.receivedBuf = bytes.NewBuffer(nil)
			}
		}
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

func WriteCloser(ctx context.Context) io.WriteCloser {
	if stdout := stdoutFromContext(ctx); stdout != nil {
		return stdout
	}

	return os.Stdout
}

func stdoutFromContext(ctx context.Context) *stdout {
	stdout, _ := ctx.Value(stdoutKey{}).(*stdout)
	return stdout
}
