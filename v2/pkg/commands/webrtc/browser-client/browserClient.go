package browserclient

import (
	"bytes"
	"fmt"
	"html/template"
	"log"

	"github.com/pion/webrtc/v3"
	"github.com/pkg/browser"
	signallingserver "github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/signalling-server"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/ice"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
		Short: "Get the webrtc browser client",
		Long:  "Get the webrtc browser client",
		RunE:  c.run,
	}

	flags := c.cobraCommand.Flags()
	flags.Bool("open", false, "Open the client in the browser automatically")

	return c.cobraCommand
}

func (c *Cmd) run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	output.InvocationInfo(ctx, cmd, args)
	defer func() {
		events.Succeeded(ctx)
		events.Stop(ctx)
	}()

	if err := c.configureWebRTC(cmd.Flags()); err != nil {
		return err
	}

	t, err := template.New("root").Parse(signallingserver.HTMLTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}
	buf := bytes.NewBuffer(nil)
	err = t.Execute(buf, map[string]any{
		"AutoConnect":  false,
		"ClientJS":     template.HTML(signallingserver.BrowserClientJS),
		"ICEServerURL": c.webrtcConfig.ICEServers[0].URLs[0],
		"SessionID":    0,
		"Offer":        "",
	})

	openBrowser, _ := cmd.Flags().GetBool("open")
	if openBrowser {
		if err := browser.OpenReader(buf); err != nil {
			log.Println("failed to open browser:", err)
		}
	} else {
		fmt.Print(buf.String())
	}

	return err
}

func (c *Cmd) configureWebRTC(flags *pflag.FlagSet) error {
	urls, _ := flags.GetStringSlice("webrtc-ice-servers")
	if len(urls) == 0 {
		urls = ice.STUNServerURLS
	}

	c.webrtcConfig = &webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: urls,
			},
		},
	}

	return nil
}
