package version

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
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
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			ofa := cmd.Flags().Lookup("output").Value.(*commands.OutputFlagArg)
			if ofa.Format == "json" {
				payload := map[string]string{}
				if version := linkersetvalues.Version; version != "" {
					payload["version"] = version
				}
				if license := linkersetvalues.License; license != "" {
					payload["license"] = license
				}
				if credit := linkersetvalues.Credit; credit != "" {
					payload["credit"] = credit
				}

				enc := json.NewEncoder(os.Stdout)

				ugly := false
				for _, opt := range ofa.Opts {
					if opt == "ugly" {
						ugly = true
					}
				}

				if !ugly {
					enc.SetIndent("", "  ")
				}

				if err := enc.Encode(payload); err != nil {
					log.Printf("error encoding json: %v", err)
				}
			} else {
				if version := linkersetvalues.Version; version != "" {
					fmt.Printf("version: %s\n", version)
				}
				if license := linkersetvalues.License; license != "" {
					fmt.Printf("license: %s\n", license)
				}
				if credit := linkersetvalues.Credit; credit != "" {
					fmt.Printf("credit: %s\n", credit)
				}
			}
		},
	}

	return c.cobraCommand
}
