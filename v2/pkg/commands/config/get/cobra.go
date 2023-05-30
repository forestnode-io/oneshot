package get

import (
	"fmt"
	"os"

	"github.com/oneshot-uno/oneshot/v2/pkg/configuration"
	"github.com/oneshot-uno/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

func New(config *configuration.Root) *Cmd {
	return &Cmd{
		config: config,
	}
}

type Cmd struct {
	cobraCommand *cobra.Command
	config       *configuration.Root
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "get path",
		Short: "Get an individual value from a oneshot configuration file.",
		Long:  "Get an individual value from a oneshot configuration file.",
		RunE:  c.run,
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)

	return c.cobraCommand
}

func (c *Cmd) run(cmd *cobra.Command, args []string) error {
	var (
		node yaml.Node
		err  error
	)

	if len(args) < 1 {
		return output.UsageErrorF("not enough arguments")
	}

	if configuration.ConfigPath == "" {
		return output.UsageErrorF("no configuration file specified")
	}

	fileBytes, err := os.ReadFile(configuration.ConfigPath)
	if err != nil {
		return output.UsageErrorF("failed to read configuration file: %w", err)
	}

	if err := yaml.Unmarshal(fileBytes, &node); err != nil {
		return output.UsageErrorF("failed to unmarshal configuration file: %w", err)
	}

	path, err := yamlpath.NewPath(args[0])
	if err != nil {
		return output.UsageErrorF("failed to parse path: %w", err)
	}

	foundNodes, err := path.Find(&node)
	if err != nil {
		return output.UsageErrorF("failed to find path in configuration file: %w", err)
	}
	if len(foundNodes) == 0 {
		return output.UsageErrorF("no nodes found for path")
	}

	selectedNode := foundNodes[0]
	removeComments(selectedNode)
	styleNode(selectedNode)

	if selectedNode.Kind == yaml.ScalarNode {
		fmt.Println(selectedNode.Value)
		return nil
	}

	return yaml.NewEncoder(os.Stdout).Encode(selectedNode)
}

func removeComments(node *yaml.Node) {
	node.HeadComment = ""
	node.FootComment = ""
	for _, n := range node.Content {
		removeComments(n)
	}
}

func styleNode(node *yaml.Node) {
	switch node.Kind {
	case yaml.ScalarNode:
	case yaml.MappingNode:
		node.Style = yaml.FoldedStyle
		for _, n := range node.Content {
			styleNode(n)
		}
	case yaml.SequenceNode:
		node.Style = yaml.FoldedStyle
		for _, n := range node.Content {
			styleNode(n)
		}
	}
}
