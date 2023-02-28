package signallingserver

import (
	"log"

	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

type Cmd struct {
	cobraCommand *cobra.Command
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

	s, err := newServer(requiredID)
	if err != nil {
		return err
	}
	if err := s.run(ctx, apiAddress, httpAddress); err != nil {
		log.Printf("error running server: %v", err)
	}

	log.Println("server stopped")

	return nil
}
