package configuration

import (
	browserclient "github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p/browser-client/configuration"
	client "github.com/raphaelreyna/oneshot/v2/pkg/commands/p2p/client/configuration"
)

type Configuration struct {
	BrowserClient *browserclient.Configuration
	Client        *client.Configuration
}

func (c *Configuration) Init() {
	if c.BrowserClient == nil {
		c.BrowserClient = &browserclient.Configuration{}
	}
	c.BrowserClient.Init()
	if c.Client == nil {
		c.Client = &client.Configuration{}
	}
	c.Client.Init()
}
