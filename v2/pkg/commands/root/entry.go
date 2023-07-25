package root

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/forestnode-io/oneshot/v2/pkg/commands"
	configcmd "github.com/forestnode-io/oneshot/v2/pkg/commands/config"
	discoveryserver "github.com/forestnode-io/oneshot/v2/pkg/commands/discovery-server"
	"github.com/forestnode-io/oneshot/v2/pkg/commands/exec"
	p2p "github.com/forestnode-io/oneshot/v2/pkg/commands/p2p"
	"github.com/forestnode-io/oneshot/v2/pkg/commands/receive"
	"github.com/forestnode-io/oneshot/v2/pkg/commands/redirect"
	"github.com/forestnode-io/oneshot/v2/pkg/commands/rproxy"
	"github.com/forestnode-io/oneshot/v2/pkg/commands/send"
	"github.com/forestnode-io/oneshot/v2/pkg/commands/version"
	"github.com/forestnode-io/oneshot/v2/pkg/configuration"
	"github.com/forestnode-io/oneshot/v2/pkg/events"
	"github.com/forestnode-io/oneshot/v2/pkg/flagargs"
	oneshothttp "github.com/forestnode-io/oneshot/v2/pkg/net/http"
	"github.com/forestnode-io/oneshot/v2/pkg/output"
	"github.com/pion/webrtc/v3"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type rootCommand struct {
	cobra.Command
	server     *oneshothttp.Server
	closers    []io.Closer
	middleware oneshothttp.Middleware

	outFlag flagargs.OutputFormat

	webrtcConfig *webrtc.Configuration

	handler http.HandlerFunc

	config *configuration.Root

	wg sync.WaitGroup
}

func ExecuteContext(ctx context.Context) error {
	// template funcs need to be added before any commands are created
	// since they register usage templates
	cobra.AddTemplateFunc("wrappedFlagUsages", wrappedFlagUsages)
	cobra.AddTemplateFunc("indent", func(p int, s string) string {
		padding := strings.Repeat(" ", p)
		return padding + strings.ReplaceAll(s, "\n", "\n"+padding)
	})

	var (
		root rootCommand
		cmd  = &root.Command
		err  error
	)
	root.Use = "oneshot"
	root.SilenceUsage = true
	root.PersistentPreRunE = root.init
	root.PersistentPostRunE = handleUsageErrors(
		func() { cmd.SilenceUsage = false },
		root.errorSuppressor(root.runServer),
	)
	root.config = configuration.EmptyRoot()
	root.config.Init(cmd)

	root.setSubCommands()

	ctx = commands.WithHTTPHandlerFuncSetter(ctx, &root.handler)
	ctx = commands.WithClosers(ctx, &root.closers)

	events.RegisterEventListener(ctx, output.SetEventsChan)

	root.SetHelpTemplate(helpTemplate)
	root.SetUsageTemplate(usageTemplate)

	err = root.ExecuteContext(ctx)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).
			Msg("failed to execute root command")
	}
	for _, closer := range root.closers {
		closer.Close()
	}

	return err
}

func CobraCommand(init bool) *cobra.Command {
	var (
		root rootCommand
		cmd  = root.Command
	)

	root.Use = "oneshot"
	root.config = configuration.EmptyRoot()

	root.config.Init(&cmd)

	root.setSubCommands()

	root.SetHelpTemplate(helpTemplate)
	root.SetUsageTemplate(usageTemplate)

	cobra.AddTemplateFunc("wrappedFlagUsages", wrappedFlagUsages)
	cobra.AddTemplateFunc("indent", func(p int, s string) string {
		padding := strings.Repeat(" ", p)
		return padding + strings.ReplaceAll(s, "\n", "\n"+padding)
	})

	return &root.Command
}

func (r *rootCommand) setSubCommands() {
	for _, sc := range subCommands(r.config) {
		if reFunc := sc.RunE; reFunc != nil {
			sc.RunE = func(cmd *cobra.Command, args []string) error {
				output.InvocationInfo(cmd.Context(), cmd, args)
				return reFunc(cmd, args)
			}
		} else if rFunc := sc.Run; rFunc != nil {
			sc.Run = func(cmd *cobra.Command, args []string) {
				output.InvocationInfo(cmd.Context(), cmd, args)
				rFunc(cmd, args)
			}
		}
		sc.Flags().BoolP("help", "h", false, "Show this help message.")
		r.AddCommand(sc)
	}
}

func subCommands(config *configuration.Root) []*cobra.Command {
	return []*cobra.Command{
		configcmd.New(config).Cobra(),
		exec.New(config).Cobra(),
		receive.New(config).Cobra(),
		redirect.New(config).Cobra(),
		send.New(config).Cobra(),
		rproxy.New(config).Cobra(),
		p2p.New(config).Cobra(),
		discoveryserver.New(config).Cobra(),
		version.New().Cobra(),
	}
}
