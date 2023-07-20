package configuration

import (
	browserclient "github.com/oneshot-uno/oneshot/v2/pkg/commands/p2p/browser-client/configuration"
	client "github.com/oneshot-uno/oneshot/v2/pkg/commands/p2p/client/configuration"
)

type Configuration struct {
	BrowserClient *browserclient.Configuration `mapstructure:"browserclient" yaml:"browserclient"`
	Client        *client.Configuration        `mapstructure:"client" yaml:"client"`
}
