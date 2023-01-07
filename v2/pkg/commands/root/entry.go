package root

import (
	"context"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/exec"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/receive"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/redirect"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/send"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/out"
)

func ExecuteContext(ctx context.Context) error {
	var root rootCommand
	root.Use = "oneshot"
	root.PersistentPreRunE = root.persistentPreRunE
	root.PersistentPostRunE = root.persistentPostRunE

	root.setFlags()
	root.AddCommand(exec.New().Cobra())
	root.AddCommand(receive.New().Cobra())
	root.AddCommand(redirect.New().Cobra())
	root.AddCommand(send.New().Cobra())

	ctx = events.WithEvents(ctx)
	ctx = out.WithOut(ctx)
	ctx = commands.WithHTTPHandlerFuncSetter(ctx, &root.handler)
	ctx = commands.WithClosers(ctx, &root.closers)

	events.RegisterEventListener(ctx, out.SetEventsChan)

	defer func() {
		for _, closer := range root.closers {
			closer.Close()
		}
	}()

	return root.ExecuteContext(ctx)
}
