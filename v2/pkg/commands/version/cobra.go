package version

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/forestnode-io/oneshot/v2/pkg/flagargs"
	"github.com/forestnode-io/oneshot/v2/pkg/version"
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
			ofa := cmd.Flags().Lookup("output").Value.(*flagargs.OutputFormat)
			if ofa.Format == "json" {
				payload := map[string]string{}
				if ver := version.Version; ver != "" {
					payload["version"] = ver
				}
				if apiVersion := version.Version; apiVersion != "" {
					payload["apiVersion"] = apiVersion
				}
				if license := version.License; license != "" {
					payload["license"] = license
				}
				if credit := version.Credit; credit != "" {
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
				if ver := version.Version; ver != "" {
					fmt.Printf("version: %s\n", ver)
				}
				if apiVersion := version.APIVersion; apiVersion != "" {
					fmt.Printf("api-version: %s\n", apiVersion)
				}
				if license := version.License; license != "" {
					fmt.Printf("license: %s\n", license)
				}
				if credit := version.Credit; credit != "" {
					fmt.Printf("credit: %s\n", credit)
				}
			}
		},
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)

	return c.cobraCommand
}
