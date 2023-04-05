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

	fs *pflag.FlagSet
}

func (c *Configuration) Init() {
	c.fs = pflag.NewFlagSet("discovery server configuration flags", pflag.ExitOnError)

	if c.RequiredKey == nil {
		c.RequiredKey = &Secret{}
	}
	if c.JWT == nil {
		c.JWT = &Secret{}
	}
	if c.URLAssignment == nil {
		c.URLAssignment = &URLAssignment{}
	}
	c.URLAssignment.init()
	if c.APIServer == nil {
		c.APIServer = &Server{}
	}
	c.APIServer.init()

	c.fs.String("discovery-server-required-key-path", "", "Path to the file containing the required key.")
	c.fs.String("discovery-server-required-key", "", "Value of the required key.")
	c.fs.String("discovery-server-jwt-path", "", "Path to the file containing the JWT secret.")
	c.fs.String("discovery-server-jwt", "", "Value of the JWT secret.")
	c.fs.Int("discovery-server-max-client-queue-size", 0, "Maximum number of clients that can be queued before new clients are rejected.")
	c.fs.String("discovery-server-webrtc-configuration-path", "", "Path to the file containing the WebRTC configuration.")

	c.URLAssignment.init()
	c.APIServer.init()
}

func (c *Configuration) SetFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
	c.URLAssignment.setFlags(cmd, fs)
	c.APIServer.setFlags(cmd, fs)
}

func (c *Configuration) MergeFlags() {
	if c.fs.Changed("discovery-server-required-key-path") {
		c.RequiredKey.Path, _ = c.fs.GetString("discovery-server-required-key-path")
	}
	if c.fs.Changed("discovery-server-required-key") {
		c.RequiredKey.Value, _ = c.fs.GetString("discovery-server-required-key")
	}
	if c.fs.Changed("discovery-server-jwt-path") {
		c.JWT.Path, _ = c.fs.GetString("discovery-server-jwt-path")
	}
	if c.fs.Changed("discovery-server-jwt") {
		c.JWT.Value, _ = c.fs.GetString("discovery-server-jwt")
	}
	if c.fs.Changed("discovery-server-max-client-queue-size") {
		c.MaxClientQueueSize, _ = c.fs.GetInt("discovery-server-max-client-queue-size")
	}

	c.URLAssignment.mergeFlags()
	c.APIServer.mergeFlags()
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
	if err := c.URLAssignment.hydrate(); err != nil {
		return fmt.Errorf("failed to hydrate URL assignment: %w", err)
	}
	if err := c.APIServer.hydrate(); err != nil {
		return fmt.Errorf("failed to hydrate API server: %w", err)
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

	fs *pflag.FlagSet
}

func (c *URLAssignment) init() {
	c.fs = pflag.NewFlagSet("url assignment flags", pflag.ExitOnError)

	c.fs.String("discovery-server-url-assignment-scheme", "", "URL scheme to use for the discovery server.")
	c.fs.String("discovery-server-url-assignment-domain", "", "URL domain to use for the discovery server.")
	c.fs.Int("discovery-server-url-assignment-port", 0, "URL port to use for the discovery server.")
	c.fs.String("discovery-server-url-assignment-path", "", "URL path to use for the discovery server.")
	c.fs.String("discovery-server-url-assignment-path-prefix", "", "URL path prefix to use for the discovery server.")

	cobra.AddTemplateFunc("urlAssignmentFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *URLAssignment) setFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *URLAssignment) mergeFlags() {
	if c.fs.Changed("discovery-server-url-assignment-scheme") {
		c.Scheme, _ = c.fs.GetString("discovery-server-url-assignment-scheme")
	}
	if c.fs.Changed("discovery-server-url-assignment-domain") {
		c.Domain, _ = c.fs.GetString("discovery-server-url-assignment-domain")
	}
	if c.fs.Changed("discovery-server-url-assignment-port") {
		c.Port, _ = c.fs.GetInt("discovery-server-url-assignment-port")
	}
	if c.fs.Changed("discovery-server-url-assignment-path") {
		c.Path, _ = c.fs.GetString("discovery-server-url-assignment-path")
	}
	if c.fs.Changed("discovery-server-url-assignment-path-prefix") {
		c.PathPrefix, _ = c.fs.GetString("discovery-server-url-assignment-path-prefix")
	}
}

func (c *URLAssignment) validate() error {
	return nil
}

func (c *URLAssignment) hydrate() error {
	return nil
}

type Server struct {
	Addr    string `json:"addr" yaml:"addr"`
	TLSCert string `json:"tlsCert" yaml:"tlsCert"`
	TLSKey  string `json:"tlsKey" yaml:"tlsKey"`

	fs *pflag.FlagSet
}

func (c *Server) init() {
	c.fs = pflag.NewFlagSet("server flags", pflag.ExitOnError)

	c.fs.String("discovery-server-api-addr", "", "Address to listen on.")
	c.fs.String("discovery-server-api-tlsCert", "", "Path to TLS certificate.")
	c.fs.String("discovery-server-api-tlsKey", "", "Path to TLS key.")

	cobra.AddTemplateFunc("serverFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *Server) setFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
}

func (c *Server) mergeFlags() {
	if c.fs.Changed("discovery-server-api-addr") {
		c.Addr, _ = c.fs.GetString("discovery-server-api-addr")
	}
	if c.fs.Changed("discovery-server-api-tlsCert") {
		c.TLSCert, _ = c.fs.GetString("discovery-server-api-tlsCert")
	}
	if c.fs.Changed("discovery-server-api-tlsKey") {
		c.TLSKey, _ = c.fs.GetString("discovery-server-api-tlsKey")
	}
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

func (c *Server) hydrate() error {
	return nil
}
