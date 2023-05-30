package set

import (
	"fmt"
	"os"
	"strings"

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
		Use:   "set path value...",
		Short: "Set an individual value in a oneshot configuration file.",
		Long:  "Set an individual value in a oneshot configuration file.",
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

	if len(args) < 2 {
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

	switch selectedNode.Kind {
	case yaml.ScalarNode:
		selectedNode.Value = args[1]
	case yaml.SequenceNode:
		selectedNode.Content = newScalarSequenceContent(args[1:])
	case yaml.MappingNode:
		nodeKey, err := getMappingContentNodeKey(&node, selectedNode)
		if err != nil {
			return output.UsageErrorF("failed to get node key: %w", err)
		}
		nodeKey = strings.ToLower(nodeKey)
		switch {
		case strings.HasSuffix(nodeKey, "header"):
			selectedNode.Content, err = newScalarSequenceMappingContent(args[1:])
		default:
			err = fmt.Errorf("unsupported node key: %s", nodeKey)
		}
		if err != nil {
			return output.UsageErrorF("failed to parse mapping: %w", err)
		}
	}

	fileBytes, err = yaml.Marshal(&node)
	if err != nil {
		return output.UsageErrorF("failed to marshal configuration file: %w", err)
	}

	if err := os.WriteFile(configuration.ConfigPath, fileBytes, 0644); err != nil {
		return output.UsageErrorF("failed to write configuration file: %w", err)
	}

	return err
}

func newScalarSequenceContent(values []string) []*yaml.Node {
	var content []*yaml.Node

	for _, v := range values {
		content = append(content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: v,
		})
	}

	return content
}

func newScalarSequenceMappingContent(fields []string) ([]*yaml.Node, error) {
	var content []*yaml.Node

	for _, f := range fields {
		parts := strings.Split(f, "=")
		if len(parts) != 2 {
			return nil, output.UsageErrorF("invalid field: %s", f)
		}
		name := parts[0]
		value := parts[1]
		value = strings.TrimPrefix(value, "[")
		value = strings.TrimPrefix(value, "{")
		value = strings.TrimSuffix(value, "]")
		value = strings.TrimSuffix(value, "}")

		content = append(content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: name,
			},
			&yaml.Node{
				Kind:    yaml.SequenceNode,
				Content: newScalarSequenceContent(strings.Split(value, ",")),
			},
		)
	}

	return content, nil
}

func getMappingContentNodeKey(parent, node *yaml.Node) (string, error) {
	if parent.Kind != yaml.DocumentNode {
		return "", output.UsageErrorF("parent is not a document node")
	}

	for i, n := range parent.Content {
		if n == node {
			return parent.Content[i-1].Value, nil
		} else if n.Kind == yaml.MappingNode {
			key, err := _getMappingContentNodeKey(n, node)
			if err != nil {
				if err == errNodeNotFound {
					continue
				}
				return "", err
			}
			return key, nil
		}
	}

	return "", output.UsageErrorF("node not found in parent")
}

func _getMappingContentNodeKey(parent, node *yaml.Node) (string, error) {
	if parent.Kind != yaml.MappingNode {
		return "", output.UsageErrorF("parent is not a mapping node")
	}

	for i, n := range parent.Content {
		if n == node {
			return parent.Content[i-1].Value, nil
		} else if n.Kind == yaml.MappingNode {
			key, err := _getMappingContentNodeKey(n, node)
			if err != nil {
				if err == errNodeNotFound {
					continue
				}
				return "", err
			}
			return parent.Content[i-1].Value + "." + key, nil
		}
	}

	return "", errNodeNotFound
}

var errNodeNotFound = fmt.Errorf("node not found")
