package configuration

import (
	"fmt"

	discoveryserver "github.com/oneshot-uno/oneshot/v2/pkg/commands/discovery-server/configuration"
	exec "github.com/oneshot-uno/oneshot/v2/pkg/commands/exec/configuration"
	p2p "github.com/oneshot-uno/oneshot/v2/pkg/commands/p2p/configuration"
	receive "github.com/oneshot-uno/oneshot/v2/pkg/commands/receive/configuration"
	redirect "github.com/oneshot-uno/oneshot/v2/pkg/commands/redirect/configuration"
	rproxy "github.com/oneshot-uno/oneshot/v2/pkg/commands/rproxy/configuration"
	send "github.com/oneshot-uno/oneshot/v2/pkg/commands/send/configuration"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

func (c *Subcommands) init() {
	if c.Receive == nil {
		c.Receive = &receive.Configuration{}
	}
	c.Receive.Init()
	if c.Send == nil {
		c.Send = &send.Configuration{}
	}
	c.Send.Init()
	if c.Exec == nil {
		c.Exec = &exec.Configuration{}
	}
	c.Exec.Init()
	if c.Redirect == nil {
		c.Redirect = &redirect.Configuration{}
	}
	c.Redirect.Init()
	if c.RProxy == nil {
		c.RProxy = &rproxy.Configuration{}
	}
	c.RProxy.Init()
	if c.P2P == nil {
		c.P2P = &p2p.Configuration{}
	}
	c.P2P.Init()
	if c.DiscoveryServer == nil {
		c.DiscoveryServer = &discoveryserver.Configuration{}
	}
	c.DiscoveryServer.Init()
}

type Root struct {
	Output       Output       `mapstructure:"output" yaml:"output"`
	Server       Server       `mapstructure:"server" yaml:"server"`
	BasicAuth    BasicAuth    `mapstructure:"basicAuth" yaml:"basicAuth"`
	CORS         CORS         `mapstructure:"cors" yaml:"cors"`
	NATTraversal NATTraversal `mapstructure:"natTraversal" yaml:"natTraversal"`
	Subcommands  *Subcommands `mapstructure:"subcommands" yaml:"subcommands"`
}

func (c *Root) Init() {
	c.Output.init()
	c.Server.init()
	c.BasicAuth.init()
	c.CORS.init()
	c.NATTraversal.init()
	c.Subcommands.init()
}

func (c *Root) SetFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	c.Output.setFlags(cmd, fs)
	c.Server.setFlags(cmd, fs)
	c.BasicAuth.setFlags(cmd, fs)
	c.CORS.setFlags(cmd, fs)
	c.NATTraversal.setFlags(cmd, fs)
}

func (c *Root) MergeFlags() {
	c.Output.mergeFlags()
	c.Server.mergeFlags()
	c.BasicAuth.mergeFlags()
	c.CORS.mergeFlags()
	c.NATTraversal.mergeFlags()
}

func (c *Root) Validate() error {
	if err := c.Output.validate(); err != nil {
		return err
	}
	if err := c.Server.validate(); err != nil {
		return err
	}
	if err := c.BasicAuth.validate(); err != nil {
		return err
	}
	if err := c.CORS.validate(); err != nil {
		return err
	}
	if err := c.NATTraversal.validate(); err != nil {
		return err
	}

	return nil
}

func (c *Root) Hydrate() error {
	if err := c.NATTraversal.hydrate(); err != nil {
		return fmt.Errorf("error hydrating NAT traversal configuration: %w", err)
	}

	if err := c.BasicAuth.hydrate(); err != nil {
		return fmt.Errorf("error hydrating basic auth configuration: %w", err)
	}

	return nil
}
