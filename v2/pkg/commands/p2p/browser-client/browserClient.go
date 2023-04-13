package browserclient

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/pion/webrtc/v3"
	"github.com/pkg/browser"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/discovery-server/template"
	"github.com/raphaelreyna/oneshot/v2/pkg/configuration"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func New(config *configuration.Root) *Cmd {
	return &Cmd{
		config: config,
	}
}

type Cmd struct {
	cobraCommand *cobra.Command
	webrtcConfig *webrtc.Configuration
	config       *configuration.Root
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
		PreRunE: func(cmd *cobra.Command, args []string) error {
			config := c.config.Subcommands.P2P.BrowserClient
			config.MergeFlags()
			if err := config.Validate(); err != nil {
				return output.UsageErrorF("invalid configuration: %w", err)
			}
			return nil
		},
		RunE: c.run,
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)

	c.config.Subcommands.P2P.BrowserClient.SetFlags(c.cobraCommand, c.cobraCommand.Flags())

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

	c.webrtcConfig, err = c.config.NATTraversal.P2P.WebRTCConfiguration.WebRTCConfiguration()
	if err != nil {
		return fmt.Errorf("unable to configure webrtc: %w", err)
	}

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
