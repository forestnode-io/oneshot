package start

import "github.com/spf13/cobra"

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
		Use:   "start",
		Short: "Start the signalling server",
		Long:  "Start the signalling server",
		RunE:  c.run,
	}

	return c.cobraCommand
}

func (c *Cmd) run(cmd *cobra.Command, args []string) error {
	return nil
}
