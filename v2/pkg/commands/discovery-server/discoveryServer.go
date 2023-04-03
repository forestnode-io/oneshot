package discoveryserver

import (
	"fmt"
	"log"
	"os"

	"github.com/pion/webrtc/v3"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
		Use:     "discovery-server",
		Aliases: []string{"discovery-server"},
		Short:   "A NAT traversal discovery server",
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
		RunE: c.run,
	}

	flags := c.cobraCommand.Flags()
	flags.String("p2p-config-file", "", "Path to a YAML file containing a WebRTC configuration")
	c.cobraCommand.MarkFlagRequired("p2p-config-file")
	c.cobraCommand.MarkFlagFilename("p2p-config-file", "yaml", "yml")

	return c.cobraCommand
}

func (c *Cmd) run(cmd *cobra.Command, args []string) error {
	var (
		ctx   = cmd.Context()
		flags = cmd.Flags()

		configFile, _ = flags.GetString("p2p-config-file")
	)

	output.InvocationInfo(ctx, cmd, args)

	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("unable to read p2p config file: %w", err)
	}
	config := Config{}
	if err = yaml.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("unable to parse p2p config file: %w", err)
	}
	c.webrtcConfig, err = config.WebRTCConfiguration.WebRTCConfiguration()
	if err != nil {
		return fmt.Errorf("unable to configure p2p: %w", err)
	}

	if c.webrtcConfig == nil {
		return fmt.Errorf("webrtc configuration is required")
	}

	s, err := newServer(&config)
	if err != nil {
		return fmt.Errorf("unable to create signalling server: %w", err)
	}
	if err := s.run(ctx); err != nil {
		log.Printf("error running server: %v", err)
	}

	log.Println("server stopped")

	return nil
}

type ClientOfferRequestResponse struct {
	RTCSessionDescription *webrtc.SessionDescription `json:"RTCSessionDescription"`
	RTCConfiguration      *webrtc.Configuration      `json:"RTCConfiguration"`
	SessionID             string                     `json:"SessionID"`
}
