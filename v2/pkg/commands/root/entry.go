package root

import (
	"context"
	"io"
	"net/http"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/exec"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/receive"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/redirect"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/send"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/version"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

type rootCommand struct {
	cobra.Command
	server     *oneshothttp.Server
	closers    []io.Closer
	middleware oneshothttp.Middleware

	outFlag outputFormatFlagArg

	handler http.HandlerFunc
}

func ExecuteContext(ctx context.Context) error {
	var (
		root rootCommand
		err  error
	)
	root.Use = "oneshot"
	root.PersistentPreRun = root.init
	root.PersistentPostRunE = root.runServer

	root.setFlags()
	root.AddCommand(exec.New().Cobra())
	root.AddCommand(receive.New().Cobra())
	root.AddCommand(redirect.New().Cobra())
	root.AddCommand(send.New().Cobra())
	root.AddCommand(version.New().Cobra())

	ctx = events.WithEvents(ctx)
	ctx, err = output.WithOutput(ctx)
	if err != nil {
		return err
	}
	ctx = commands.WithHTTPHandlerFuncSetter(ctx, &root.handler)
	ctx = commands.WithClosers(ctx, &root.closers)

	events.RegisterEventListener(ctx, output.SetEventsChan)

	err = root.ExecuteContext(ctx)
	for _, closer := range root.closers {
		closer.Close()
	}

	events.Stop(ctx)

	output.Wait(ctx)

	return err
}

func CobraCommand() *cobra.Command {
	var root rootCommand
	root.Use = "oneshot"

	root.setFlags()
	root.AddCommand(exec.New().Cobra())
	root.AddCommand(receive.New().Cobra())
	root.AddCommand(redirect.New().Cobra())
	root.AddCommand(send.New().Cobra())
	root.AddCommand(version.New().Cobra())

	return &root.Command
}
