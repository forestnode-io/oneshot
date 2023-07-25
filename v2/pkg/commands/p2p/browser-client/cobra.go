package browserclient

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/forestnode-io/oneshot/v2/pkg/commands/discovery-server/template"
	"github.com/forestnode-io/oneshot/v2/pkg/commands/p2p/browser-client/configuration"
	rootconfig "github.com/forestnode-io/oneshot/v2/pkg/configuration"
	"github.com/forestnode-io/oneshot/v2/pkg/events"
	"github.com/forestnode-io/oneshot/v2/pkg/output"
	"github.com/pion/webrtc/v3"
	"github.com/pkg/browser"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func New(config *rootconfig.Root) *Cmd {
	return &Cmd{
		config: config,
	}
}

type Cmd struct {
	cobraCommand *cobra.Command
	webrtcConfig *webrtc.Configuration
	config       *rootconfig.Root
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "browser-client",
		Short: "Get the p2p browser client as a single HTML file.",
		Long: `Get the p2p browser client as a single HTML file.
This client can be used to establish a p2p connection with oneshot when not using a discovery server.`,
		RunE: c.run,
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)
	configuration.SetFlags(c.cobraCommand)

	return c.cobraCommand
}

func (c *Cmd) run(cmd *cobra.Command, args []string) error {
	var (
		ctx    = cmd.Context()
		log    = zerolog.Ctx(ctx)
		config = c.config.Subcommands.P2P.BrowserClient
		err    error
	)

	output.InvocationInfo(ctx, cmd, args)
	defer func() {
		events.Succeeded(ctx)
		events.Stop(ctx)
	}()

	p2pConfig := c.config.NATTraversal.P2P
	iwc, err := p2pConfig.ParseConfig()
	if err != nil {
		return fmt.Errorf("failed to parse p2p configuration: %w", err)
	}
	wc, err := iwc.WebRTCConfiguration()
	if err != nil {
		return fmt.Errorf("failed to get WebRTC configuration: %w", err)
	}

	c.webrtcConfig = wc

	rtcConfigJSON, err := json.Marshal(c.webrtcConfig)
	if err != nil {
		return fmt.Errorf("unable to marshal webrtc config: %w", err)
	}

	tmpltCtx := template.Context{
		AutoConnect:   false,
		ClientJS:      template.ClientJS,
		PolyfillJS:    template.PolyfillJS,
		RTCConfigJSON: string(rtcConfigJSON),
	}
	buf := bytes.NewBuffer(nil)
	err = template.WriteTo(buf, tmpltCtx)
	if err != nil {
		return fmt.Errorf("unable to write template: %w", err)
	}

	if config.Open {
		if err := browser.OpenReader(buf); err != nil {
			log.Error().Err(err).
				Msg("failed to open browser")
		}
	} else {
		fmt.Print(buf.String())
	}

	return err
}
