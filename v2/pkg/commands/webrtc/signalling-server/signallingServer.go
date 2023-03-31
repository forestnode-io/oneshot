package signallingserver

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
		Use:     "signalling-server",
		Aliases: []string{"signaling-server"},
		Short:   "WebRTC signalling server",
		Long:    "WebRTC signalling server",
		SuggestFor: []string{
			"browser-client",
			"client send",
			"client receive",
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
