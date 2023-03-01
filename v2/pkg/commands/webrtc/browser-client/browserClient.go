package browserclient

import (
	"bytes"
	"fmt"
	"html/template"
	"log"

	"github.com/pkg/browser"
	signallingserver "github.com/raphaelreyna/oneshot/v2/pkg/commands/webrtc/signalling-server"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

func New() *Cmd {
	return &Cmd{}
}

type Cmd struct {
	cobraCommand *cobra.Command
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

	t, err := template.New("root").Parse(signallingserver.HTMLTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}
	buf := bytes.NewBuffer(nil)
	err = t.Execute(buf, map[string]any{
		"AutoConnect":  false,
		"ClientJS":     template.HTML(signallingserver.BrowserClientJS),
		"ICEServerURL": "stun:stun.l.google.com:19302",
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
