package signallingserver

import (
	"log"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/ice"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Cmd struct {
	cobraCommand *cobra.Command
	webrtcConfig *webrtc.Configuration
}

func New() *Cmd {
	return &Cmd{}
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:     "signalling-server",
		Aliases: []string{"signaling-server"},
		Short:   "WebRTC signalling server",
		Long:    "WebRTC signalling server",
		RunE:    c.run,
	}

	flags := c.cobraCommand.Flags()
	flags.String("http-address", ":8080", "Address to listen on for HTTP requests")
	flags.String("api-address", ":8081", "Address to listen on for API requests from oneshot servers")
	flags.String("required-id", "", "Required ID for clients to connect to this server")

	return c.cobraCommand
}

func (c *Cmd) run(cmd *cobra.Command, args []string) error {
	var (
		ctx   = cmd.Context()
		flags = cmd.Flags()

		httpAddress, _ = flags.GetString("http-address")
		apiAddress, _  = flags.GetString("api-address")
		requiredID, _  = flags.GetString("required-id")
	)

	output.InvocationInfo(ctx, cmd, args)

	if err := c.configureWebRTC(flags); err != nil {
		return err
	}

	iceServerURL := c.webrtcConfig.ICEServers[0].URLs[0]
	s, err := newServer(iceServerURL, requiredID)
	if err != nil {
		return err
	}
	if err := s.run(ctx, apiAddress, httpAddress); err != nil {
		log.Printf("error running server: %v", err)
	}

	log.Println("server stopped")

	return nil
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
