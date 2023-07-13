package configuration

import (
	"fmt"

	discoveryserver "github.com/oneshot-uno/oneshot/v2/pkg/commands/discovery-server/configuration"
	exec "github.com/oneshot-uno/oneshot/v2/pkg/commands/exec/configuration"
	browserclient "github.com/oneshot-uno/oneshot/v2/pkg/commands/p2p/browser-client/configuration"
	client "github.com/oneshot-uno/oneshot/v2/pkg/commands/p2p/client/configuration"
	clientreceive "github.com/oneshot-uno/oneshot/v2/pkg/commands/p2p/client/receive/configuration"
	clientsend "github.com/oneshot-uno/oneshot/v2/pkg/commands/p2p/client/send/configuration"
	p2p "github.com/oneshot-uno/oneshot/v2/pkg/commands/p2p/configuration"
	receive "github.com/oneshot-uno/oneshot/v2/pkg/commands/receive/configuration"
	redirect "github.com/oneshot-uno/oneshot/v2/pkg/commands/redirect/configuration"
	rproxy "github.com/oneshot-uno/oneshot/v2/pkg/commands/rproxy/configuration"
	send "github.com/oneshot-uno/oneshot/v2/pkg/commands/send/configuration"
	"github.com/spf13/cobra"
)

type Subcommands struct {
	Receive         *receive.Configuration         `mapstructure:"receive" yaml:"receive"`
	Send            *send.Configuration            `mapstructure:"send" yaml:"send"`
	Exec            *exec.Configuration            `mapstructure:"exec" yaml:"exec"`
	Redirect        *redirect.Configuration        `mapstructure:"redirect" yaml:"redirect"`
	RProxy          *rproxy.Configuration          `mapstructure:"rproxy" yaml:"rproxy"`
	P2P             *p2p.Configuration             `mapstructure:"p2p" yaml:"p2p"`
	DiscoveryServer *discoveryserver.Configuration `mapstructure:"discoveryServer" yaml:"discoveryServer"`
}

func (c *Subcommands) init(cmd *cobra.Command) {
	if c.Receive == nil {
		c.Receive = &receive.Configuration{}
	}
	if c.Send == nil {
		c.Send = &send.Configuration{}
	}
	if c.Exec == nil {
		c.Exec = &exec.Configuration{}
	}
	if c.Redirect == nil {
		c.Redirect = &redirect.Configuration{}
	}
	if c.RProxy == nil {
		c.RProxy = &rproxy.Configuration{}
	}
	if c.P2P == nil {
		c.P2P = &p2p.Configuration{
			BrowserClient: &browserclient.Configuration{},
			Client: &client.Configuration{
				Receive: &clientreceive.Configuration{},
				Send:    &clientsend.Configuration{},
			},
		}
	}
	if c.DiscoveryServer == nil {
		c.DiscoveryServer = &discoveryserver.Configuration{}
	}
}

func (s *Subcommands) validate() error {
	if err := s.Receive.Validate(); err != nil {
		return fmt.Errorf("error validating receive configuration: %w", err)
	}
	if err := s.Send.Validate(); err != nil {
		return fmt.Errorf("error validating send configuration: %w", err)
	}
	if err := s.Exec.Validate(); err != nil {
		return fmt.Errorf("error validating exec configuration: %w", err)
	}
	if err := s.Redirect.Validate(); err != nil {
		return fmt.Errorf("error validating redirect configuration: %w", err)
	}
	if err := s.RProxy.Validate(); err != nil {
		return fmt.Errorf("error validating rproxy configuration: %w", err)
	}
	if err := s.DiscoveryServer.Validate(); err != nil {
		return fmt.Errorf("error validating discovery server configuration: %w", err)
	}

	return nil
}

func (s *Subcommands) hydrate() error {
	if err := s.Receive.Hydrate(); err != nil {
		return fmt.Errorf("error hydrating receive configuration: %w", err)
	}
	if err := s.Send.Hydrate(); err != nil {
		return fmt.Errorf("error hydrating send configuration: %w", err)
	}
	if err := s.Exec.Hydrate(); err != nil {
		return fmt.Errorf("error hydrating exec configuration: %w", err)
	}
	if err := s.Redirect.Hydrate(); err != nil {
		return fmt.Errorf("error hydrating redirect configuration: %w", err)
	}
	if err := s.RProxy.Hydrate(); err != nil {
		return fmt.Errorf("error hydrating rproxy configuration: %w", err)
	}
	if err := s.DiscoveryServer.Hydrate(); err != nil {
		return fmt.Errorf("error hydrating discovery server configuration: %w", err)
	}

	return nil
}

type Root struct {
	Output       Output       `mapstructure:"output" yaml:"output"`
	Server       Server       `mapstructure:"server" yaml:"server"`
	BasicAuth    BasicAuth    `mapstructure:"basicAuth" yaml:"basicAuth"`
	CORS         CORS         `mapstructure:"cors" yaml:"cors"`
	NATTraversal NATTraversal `mapstructure:"natTraversal" yaml:"natTraversal"`
	Subcommands  *Subcommands `mapstructure:"cmd" yaml:"cmd"`
	Discovery    Discovery    `mapstructure:"discovery" yaml:"discovery"`
}

func EmptyRoot() *Root {
	return &Root{
		Subcommands: &Subcommands{
			Receive:  &receive.Configuration{},
			Send:     &send.Configuration{},
			Exec:     &exec.Configuration{},
			Redirect: &redirect.Configuration{},
			RProxy:   &rproxy.Configuration{},
			P2P: &p2p.Configuration{
				BrowserClient: &browserclient.Configuration{},
				Client: &client.Configuration{
					Receive: &clientreceive.Configuration{},
					Send:    &clientsend.Configuration{},
				},
			},
			DiscoveryServer: &discoveryserver.Configuration{},
		},
	}
}

func (c *Root) Init(cmd *cobra.Command) {
	setOutputFlags(cmd)
	setServerFlags(cmd)
	setBasicAuthFlags(cmd)
	setCORSFlags(cmd)
	setNATTraversalFlags(cmd)
	setDiscoveryFlags(cmd)
	c.Subcommands = &Subcommands{}
	c.Subcommands.init(cmd)
}

func (c *Root) Validate() error {
	if err := c.Subcommands.validate(); err != nil {
		return fmt.Errorf("error validating subcommands: %w", err)
	}

	if err := c.Output.validate(); err != nil {
		return fmt.Errorf("error validating output configuration: %w", err)
	}

	if err := c.Server.validate(); err != nil {
		return fmt.Errorf("error validating server configuration: %w", err)
	}

	if err := c.BasicAuth.validate(); err != nil {
		return fmt.Errorf("error validating basic auth configuration: %w", err)
	}

	if err := c.CORS.validate(); err != nil {
		return fmt.Errorf("error validating CORS configuration: %w", err)
	}

	if err := c.NATTraversal.validate(); err != nil {
		return fmt.Errorf("error validating NAT traversal configuration: %w", err)
	}

	return nil
}

func (c *Root) Hydrate() error {
	if err := c.Subcommands.hydrate(); err != nil {
		return fmt.Errorf("error hydrating subcommands: %w", err)
	}

	if err := c.NATTraversal.hydrate(); err != nil {
		return fmt.Errorf("error hydrating NAT traversal configuration: %w", err)
	}

	if err := c.BasicAuth.hydrate(); err != nil {
		return fmt.Errorf("error hydrating basic auth configuration: %w", err)
	}

	if err := c.Discovery.hydrate(); err != nil {
		return fmt.Errorf("error hydrating discovery configuration: %w", err)
	}

	return nil
}
