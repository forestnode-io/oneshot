package discoveryserver

import (
	"fmt"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/configuration"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type Cmd struct {
	cobraCommand *cobra.Command
	config       *configuration.Root
}

func New(config *configuration.Root) *Cmd {
	return &Cmd{
		config: config,
	}
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "discovery-server",
		Short: "A NAT traversal discovery server",
		Long: `A NAT traversal discovery server.
If using UPnP-IGD NAT traversal, the discovery server will redirect traffic to the public ip + newly opened external port.
This allows for a dynamic DNS type service.
If using P2P NAT traversal, the discovery server will act as the signalling server for the peers to establish a connection.
The discovery server will accept both other oneshot instances and web browsers as clients.
Web browsers will be served a JS WebRTC client that will connect back to the discovery server and perform the P2P NAT traversal.
`,
		SuggestFor: []string{
			"p2p browser-client",
			"p2p client send",
			"p2p client receive",
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			config := c.config.Subcommands.DiscoveryServer
			config.MergeFlags()
			if err := config.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}
			if err := config.Hydrate(); err != nil {
				return fmt.Errorf("failed to hydrate configuration: %w", err)
			}
			return nil
		},
		RunE: c.run,
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)

	c.config.Subcommands.DiscoveryServer.SetFlags(c.cobraCommand, c.cobraCommand.LocalFlags())

	return c.cobraCommand
}

func (c *Cmd) run(cmd *cobra.Command, args []string) error {
	var (
		ctx = cmd.Context()
		log = zerolog.Ctx(ctx)
	)

	s, err := newServer(c.config)
	if err != nil {
		return fmt.Errorf("unable to create signalling server: %w", err)
	}
	if err := s.run(ctx); err != nil {
		log.Error().Err(err).
			Msg("error running server")
	}

	log.Info().Msg("discovery server exiting")

	return nil
}

type ClientOfferRequestResponse struct {
	RTCSessionDescription *webrtc.SessionDescription `json:"RTCSessionDescription"`
	RTCConfiguration      *webrtc.Configuration      `json:"RTCConfiguration"`
	SessionID             string                     `json:"SessionID"`
}
