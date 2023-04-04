package browserclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pion/webrtc/v3"
	"github.com/pkg/browser"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/discovery-server/template"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshotwebrtc "github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func New() *Cmd {
	return &Cmd{}
}

type Cmd struct {
	cobraCommand *cobra.Command
	webrtcConfig *webrtc.Configuration
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

	flags := c.cobraCommand.Flags()
	flags.Bool("open", false, "Open the client in the browser automatically")

	return c.cobraCommand
}

func (c *Cmd) run(cmd *cobra.Command, args []string) error {
	var (
		ctx = cmd.Context()
		log = zerolog.Ctx(ctx)
	)

	output.InvocationInfo(ctx, cmd, args)
	defer func() {
		events.Succeeded(ctx)
		events.Stop(ctx)
	}()

	if err := c.configureWebRTC(cmd.Flags()); err != nil {
		return err
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

	openBrowser, _ := cmd.Flags().GetBool("open")
	if openBrowser {
		if err := browser.OpenReader(buf); err != nil {
			log.Error().Err(err).
				Msg("failed to open browser")
		}
	} else {
		fmt.Print(buf.String())
	}

	return err
}

func (c *Cmd) configureWebRTC(flags *pflag.FlagSet) error {
	path, _ := flags.GetString("webrtc-config-file")
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read webrtc config file: %w", err)
	}

	config := oneshotwebrtc.Configuration{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("unable to parse webrtc config file: %w", err)
	}

	c.webrtcConfig, err = config.WebRTCConfiguration()
	if err != nil {
		return fmt.Errorf("unable to configure webrtc: %w", err)
	}

	return nil
}
