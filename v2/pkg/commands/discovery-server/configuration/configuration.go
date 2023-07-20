package configuration

import (
	"fmt"
	"os"
)

type Configuration struct {
	RequiredKey        *Secret        `mapstructure:"requiredkey" yaml:"requiredkey"`
	JWT                *Secret        `mapstructure:"jwt" yaml:"jwt"`
	MaxClientQueueSize int            `mapstructure:"maxqueuesize" yaml:"maxqueuesize"`
	URLAssignment      *URLAssignment `mapstructure:"urlassignment" yaml:"urlassignment"`
	APIServer          *Server        `mapstructure:"server" yaml:"server"`
}

func (c *Configuration) Validate() error {
	if err := c.URLAssignment.validate(); err != nil {
		return fmt.Errorf("invalid URL assignment: %w", err)
	}
	if err := c.APIServer.validate(); err != nil {
		return fmt.Errorf("invalid API server: %w", err)
	}
	return nil
}

func (c *Configuration) Hydrate() error {
	if err := c.RequiredKey.hydrate(); err != nil {
		return fmt.Errorf("failed to hydrate required key: %w", err)
	}
	if err := c.JWT.hydrate(); err != nil {
		return fmt.Errorf("failed to hydrate JWT: %w", err)
	}
	return nil
}

type Secret struct {
	Path  string `mapstructure:"path" yaml:"path"`
	Value string `mapstructure:"value" yaml:"value"`
}

func (s *Secret) hydrate() error {
	if s.Value != "" {
		return nil
	}
	if s.Path == "" {
		return nil
	}

	data, err := os.ReadFile(s.Path)
	if err != nil {
		return fmt.Errorf("failed to read secret from file: %w", err)
	}

	s.Value = string(data)

	return nil
}

type URLAssignment struct {
	Scheme     string `mapstructure:"scheme" yaml:"scheme"`
	Domain     string `mapstructure:"domain" yaml:"domain"`
	Port       int    `mapstructure:"port" yaml:"port"`
	Path       string `mapstructure:"path" yaml:"path"`
	PathPrefix string `mapstructure:"pathprefix" yaml:"pathprefix"`
}

func (c *URLAssignment) validate() error {
	return nil
}

type Server struct {
	Addr    string `mapstructure:"addr" yaml:"addr"`
	TLSCert string `mapstructure:"tlscert" yaml:"tlscert"`
	TLSKey  string `mapstructure:"tlskey" yaml:"tlskey"`
}

func (c *Server) validate() error {
	if c.TLSCert != "" && c.TLSKey != "" {
		if c.TLSCert == "" {
			return fmt.Errorf("tls cert path is empty")
		}
		if c.TLSKey == "" {
			return fmt.Errorf("tls key path is empty")
		}
	}
	return nil
}
