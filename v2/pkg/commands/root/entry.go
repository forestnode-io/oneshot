package root

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	discoveryserver "github.com/raphaelreyna/oneshot/v2/pkg/commands/discovery-server"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/exec"
	p2p "github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/receive"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/redirect"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/rproxy"
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

	outFlag       commands.OutputFlagArg
	externalAddrs []string

	webrtcConfig *webrtc.Configuration

	handler http.HandlerFunc

	wg sync.WaitGroup
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
	root.setSubCommands()

	ctx = commands.WithHTTPHandlerFuncSetter(ctx, &root.handler)
	ctx = commands.WithClosers(ctx, &root.closers)

	events.RegisterEventListener(ctx, output.SetEventsChan)

	root.SetHelpTemplate(helpTemplate)
	root.SetUsageTemplate(usageTemplate)

	cobra.AddTemplateFunc("wrappedFlagUsages", wrappedFlagUsages)
	cobra.AddTemplateFunc("indent", func(p int, s string) string {
		padding := strings.Repeat(" ", p)
		return padding + strings.ReplaceAll(s, "\n", "\n"+padding)
	})

	err = root.ExecuteContext(ctx)
	for _, closer := range root.closers {
		closer.Close()
	}

	return err
}

func CobraCommand() *cobra.Command {
	var root rootCommand
	root.Use = "oneshot"

	root.setFlags()
	root.setSubCommands()

	return &root.Command
}

func (r *rootCommand) setSubCommands() {
	for _, sc := range subCommands() {
		if reFunc := sc.RunE; reFunc != nil {
			sc.RunE = func(cmd *cobra.Command, args []string) error {
				output.InvocationInfo(cmd.Context(), cmd, args)
				return reFunc(cmd, args)
			}
		}
		sc.Flags().BoolP("help", "h", false, "Show this help message.")
		r.AddCommand(sc)
	}
}

func subCommands() []*cobra.Command {
	return []*cobra.Command{
		exec.New().Cobra(),
		receive.New().Cobra(),
		redirect.New().Cobra(),
		send.New().Cobra(),
		rproxy.New().Cobra(),
		p2p.New().Cobra(),
		discoveryserver.New().Cobra(),
		version.New().Cobra(),
	}
}
