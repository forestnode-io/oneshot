package configuration

import (
	receive "github.com/oneshot-uno/oneshot/v2/pkg/commands/p2p/client/receive/configuration"
	send "github.com/oneshot-uno/oneshot/v2/pkg/commands/p2p/client/send/configuration"
)

type Configuration struct {
	Receive *receive.Configuration `mapstructure:"receive" yaml:"receive"`
	Send    *send.Configuration    `mapstructure:"send" yaml:"send"`
}

func (c *Configuration) Init() {
	if c.Receive == nil {
		c.Receive = &receive.Configuration{}
	}
	c.Receive.Init()
	if c.Send == nil {
		c.Send = &send.Configuration{}
	}
	c.Send.Init()
}
