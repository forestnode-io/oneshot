package version

import (
	"fmt"

	linkersetvalues "github.com/raphaelreyna/oneshot/v2/pkg/linkerSetValues"
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
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			if version := linkersetvalues.Version; version != "" {
				fmt.Printf("version: %s\n", version)
			}
			if license := linkersetvalues.License; license != "" {
				fmt.Printf("license: %s\n", license)
			}
			if credit := linkersetvalues.Credit; credit != "" {
				fmt.Printf("credit: %s\n", credit)
			}
		},
	}

	return c.cobraCommand
}
