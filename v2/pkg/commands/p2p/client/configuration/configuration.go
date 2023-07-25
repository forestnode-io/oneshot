package configuration

import (
	receive "github.com/forestnode-io/oneshot/v2/pkg/commands/p2p/client/receive/configuration"
	send "github.com/forestnode-io/oneshot/v2/pkg/commands/p2p/client/send/configuration"
)

type Configuration struct {
	Receive *receive.Configuration `mapstructure:"receive" yaml:"receive"`
	Send    *send.Configuration    `mapstructure:"send" yaml:"send"`
}
