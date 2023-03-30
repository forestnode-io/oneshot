package signallingserver

import (
	"fmt"
	"log"
	"os"

	"github.com/pion/webrtc/v3"
	oneshotwebrtc "github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
		return fmt.Errorf("unable to configure webrtc: %w", err)
	}

	s := newServer(requiredID, c.webrtcConfig)
	if err := s.run(ctx, apiAddress, httpAddress); err != nil {
		log.Printf("error running server: %v", err)
	}

	log.Println("server stopped")

	return nil
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

type ClientOfferRequestResponse struct {
	RTCSessionDescription *webrtc.SessionDescription `json:"RTCSessionDescription"`
	RTCConfiguration      *webrtc.Configuration      `json:"RTCConfiguration"`
	SessionID             string                     `json:"SessionID"`
}
