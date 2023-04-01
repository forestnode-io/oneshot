package signallingserver

import (
	"fmt"

	network "github.com/raphaelreyna/oneshot/v2/pkg/net"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc"
)

type JWTSecretConfig struct {
	Path  string `yaml:"path" mapstructure:"path"`
	Value string `yaml:"value" mapstructure:"value"`
}

type RequiredID struct {
	Path  string `yaml:"path" mapstructure:"path"`
	Value string `yaml:"value" mapstructure:"value"`
}

type URLAssignment struct {
	Scheme     string `yaml:"scheme" mapstructure:"scheme"`
	Domain     string `yaml:"domain" mapstructure:"domain"`
	Port       int    `yaml:"port" mapstructure:"port"`
	Path       string `yaml:"path" mapstructure:"path"`
	PathPrefix string `yaml:"pathPrefix" mapstructure:"pathPrefix"`
}

type TLS struct {
	Cert string `yaml:"cert" mapstructure:"cert"`
	Key  string `yaml:"key" mapstructure:"key"`
}

type Server struct {
	Addr string `yaml:"addr" mapstructure:"addr"`
	TLS  *TLS   `yaml:"tls" mapstructure:"tls"`
}

type Servers struct {
	HTTP      Server `yaml:"http" mapstructure:"http"`
	Discovery Server `yaml:"discovery" mapstructure:"discovery"`
}

type Config struct {
	Servers             Servers              `yaml:"servers" mapstructure:"servers"`
	URLAssignment       URLAssignment        `yaml:"urlAssignment" mapstructure:"urlAssignment"`
	RequiredID          RequiredID           `yaml:"requiredID" mapstructure:"requiredID"`
	MaxClientQueueSize  int                  `yaml:"maxClientQueueSize" mapstructure:"maxClientQueueSize"`
	JWTSecretConfig     JWTSecretConfig      `yaml:"jwt" mapstructure:"jwt"`
	WebRTCConfiguration webrtc.Configuration `yaml:"p2p" mapstructure:"p2p"`
}

func (c *Config) SetDefaults() error {
	if c.MaxClientQueueSize == 0 {
		c.MaxClientQueueSize = 10
	}
	if c.URLAssignment.Domain == "" {
		ip, err := network.GetSourceIP("", 80)
		if err != nil {
			return err
		}
		c.URLAssignment.Domain = ip
	}
	if c.URLAssignment.Port == 0 {
		c.URLAssignment.Port = 8080
	}

	if c.JWTSecretConfig.Path == "" && c.JWTSecretConfig.Value == "" {
		return fmt.Errorf("jwt secret config must have a path or a value")
	}

	return nil
}
