package configuration

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Configuration struct {
	RequiredKey        *Secret        `json:"requiredKey" yaml:"requiredKey"`
	JWT                *Secret        `json:"jwt" yaml:"jwt"`
	MaxClientQueueSize int            `json:"maxClientQueueSize" yaml:"maxClientQueueSize"`
	URLAssignment      *URLAssignment `json:"urlAssignment" yaml:"urlAssignment"`
	APIServer          *Server        `json:"apiServer" yaml:"apiServer"`
}

func (c *Configuration) Init() {}

func (c *Configuration) SetFlags(cmd *cobra.Command, fs *pflag.FlagSet) {}

func (c *Configuration) MergeFlags() {}

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
	Path  string `json:"path" yaml:"path"`
	Value string `json:"value" yaml:"value"`
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
	Scheme     string `json:"scheme" yaml:"scheme"`
	Domain     string `json:"domain" yaml:"domain"`
	Port       int    `json:"port" yaml:"port"`
	Path       string `json:"path" yaml:"path"`
	PathPrefix string `json:"pathPrefix" yaml:"pathPrefix"`
}

func (c *URLAssignment) validate() error {
	return nil
}

type Server struct {
	Addr    string `json:"addr" yaml:"addr"`
	TLSCert string `json:"tlsCert" yaml:"tlsCert"`
	TLSKey  string `json:"tlsKey" yaml:"tlsKey"`
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
