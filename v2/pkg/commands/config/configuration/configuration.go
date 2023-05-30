package configuration

import (
	set "github.com/oneshot-uno/oneshot/v2/pkg/commands/config/set/configuration"
)

type Configuration struct {
	Set *set.Configuration
}

func (c *Configuration) Init() {
	if c.Set == nil {
		c.Set = &set.Configuration{}
	}
	c.Set.Init()
}
