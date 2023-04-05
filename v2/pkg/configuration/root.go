package configuration

import (
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/exec"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/receive"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/redirect"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/rproxy"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands/send"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Subcommands struct {
	Receive  receive.Configuration  `mapstructure:"receive" yaml:"receive"`
	Send     send.Configuration     `mapstructure:"send" yaml:"send"`
	Exec     exec.Configuration     `mapstructure:"exec" yaml:"exec"`
	Redirect redirect.Configuration `mapstructure:"redirect" yaml:"redirect"`
	RProxy   rproxy.Configuration   `mapstructure:"rproxy" yaml:"rproxy"`
	P2P      p2p.Configuration      `mapstructure:"p2p" yaml:"p2p"`
}

func (c *Subcommands) init() {
	c.Receive.Init()
}

type Root struct {
	Output       Output       `mapstructure:"output" yaml:"output"`
	Server       Server       `mapstructure:"server" yaml:"server"`
	BasicAuth    BasicAuth    `mapstructure:"basicAuth" yaml:"basicAuth"`
	CORS         CORS         `mapstructure:"cors" yaml:"cors"`
	NATTraversal NATTraversal `mapstructure:"natTraversal" yaml:"natTraversal"`
	Subcommands  Subcommands  `mapstructure:"subcommands" yaml:"subcommands"`
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
