package configuration

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type NATTraversal struct {
	DiscoveryServer DiscoveryServer `mapstructure:"discoveryServer" yaml:"discoveryServer"`
	P2P             P2P             `mapstructure:"p2p" yaml:"p2p"`
	UPnP            UPnP            `mapstructure:"upnp" yaml:"upnp"`
}

func (c *NATTraversal) init() {
	c.DiscoveryServer.init()
	c.P2P.init()
	c.UPnP.init()
}

func (c *NATTraversal) setFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	c.DiscoveryServer.setFlags(cmd, fs)
	c.P2P.setFlags(cmd, fs)
	c.UPnP.setFlags(cmd, fs)

	cmd.MarkFlagsMutuallyExclusive("p2p-discovery-dir",
		"discovery-server-url",
		"discovery-server-key-path",
		"discovery-server-key",
		"discovery-server-insecure",
		"discovery-server-preferred-url",
		"discovery-server-required-url",
	)
}

func (c *NATTraversal) mergeFlags() {
	c.DiscoveryServer.mergeFlags()
	c.P2P.mergeFlags()
	c.UPnP.mergeFlags()
}

func (c *NATTraversal) validate() error {
	if err := c.DiscoveryServer.validate(); err != nil {
		return fmt.Errorf("invalid discovery server configuration: %w", err)
	}
	if err := c.P2P.validate(); err != nil {
		return fmt.Errorf("invalid P2P configuration: %w", err)
	}
	if err := c.UPnP.validate(); err != nil {
		return fmt.Errorf("invalid UPnP configuration: %w", err)
	}

	return nil
}

func (c *NATTraversal) hydrate() error {
	if err := c.P2P.hydrate(); err != nil {
		return fmt.Errorf("failed to hydrate P2P configuration: %w", err)
	}
	return nil
}

type DiscoveryServer struct {
	URL          string `mapstructure:"url" yaml:"url"`
	KeyPath      string `mapstructure:"keyPath" yaml:"keyPath"`
	Key          string `mapstructure:"key" yaml:"key,omitempty"`
	Insecure     bool   `mapstructure:"insecure" yaml:"insecure"`
	PreferredURL string `mapstructure:"preferredURL" yaml:"preferredURL"`
	RequiredURL  string `mapstructure:"requiredURL" yaml:"requiredURL"`

	fs *pflag.FlagSet
}

func (c *DiscoveryServer) init() {
	c.fs = pflag.NewFlagSet("Discovery Server", pflag.ExitOnError)

	c.fs.String("discovery-server-url", "", "URL of the discovery server to connect to.")
	c.fs.String("discovery-server-key-path", "", "Path to the key to present to the discovery server.")
	c.fs.String("discovery-server-key", "", "Key to present to the discovery server.")
	c.fs.Bool("discovery-server-insecure", false, "Allow insecure connections to the discovery server.")
	c.fs.String("discovery-server-preferred-url", "", "URL that the discovery server should try to reserve for connecting client.")
	c.fs.String("discovery-server-required-url", "", "URL that the discovery server must reserve for connecting client.")

	cobra.AddTemplateFunc("discoveryServerFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *DiscoveryServer) setFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
	cmd.MarkFlagFilename("discovery-server-key-path")
	cmd.MarkFlagsMutuallyExclusive("discovery-server-preferred-url", "discovery-server-required-url")
	cmd.MarkFlagsMutuallyExclusive("discovery-server-key-path", "discovery-server-key")
}

func (c *DiscoveryServer) mergeFlags() {
	if c.fs.Changed("discovery-server-url") {
		c.URL, _ = c.fs.GetString("discovery-server-url")
	}
	if c.fs.Changed("discovery-server-key-path") {
		c.KeyPath, _ = c.fs.GetString("discovery-server-key-path")
	}
	if c.fs.Changed("discovery-server-key") {
		c.Key, _ = c.fs.GetString("discovery-server-key")
	}
	if c.fs.Changed("discovery-server-insecure") {
		c.Insecure, _ = c.fs.GetBool("discovery-server-insecure")
	}
	if c.fs.Changed("discovery-server-preferred-url") {
		c.PreferredURL, _ = c.fs.GetString("discovery-server-preferred-url")
	}
	if c.fs.Changed("discovery-server-required-url") {
		c.RequiredURL, _ = c.fs.GetString("discovery-server-required-url")
	}
}

func (c *DiscoveryServer) validate() error {
	if c.URL == "" {
		return nil
	}

	return nil
}

type P2P struct {
	Enabled                 bool                  `mapstructure:"enabled" yaml:"enabled"`
	Only                    bool                  `mapstructure:"only" yaml:"only"`
	WebRTCConfigurationFile string                `mapstructure:"webrtcConfigurationFile" yaml:"webrtcConfigurationFile"`
	WebRTCConfiguration     *webrtc.Configuration `json:"webrtcConfiguration" yaml:"webrtcConfiguration"`
	DiscoveryDir            string                `mapstructure:"discoveryDir" yaml:"discoveryDir"`

	fs *pflag.FlagSet
}

func (c *P2P) init() {
	c.fs = pflag.NewFlagSet("P2P", pflag.ExitOnError)

	c.fs.Bool("p2p", false, `Accept incoming p2p connections.
Requires a discovery mechanism, either a discovery server or a discovery directory.`)
	c.fs.Bool("p2p-only", false, "Only accept incoming p2p connections.")
	c.fs.String("p2p-webrtc-config-file", "", `Path to the configuration file for the underlying WebRTC transport.`)
	c.fs.String("p2p-discovery-dir", "", `Path to the directory containing the discovery files.
In this directory, each peer connection has a numerically named subdirectory containing an answer and offer file.
The offer file contains the RTCSessionDescription JSON of the WebRTc offer
and the answer file contains the RTCSessionDescription JSON of the WebRTC answer.`)

	cobra.AddTemplateFunc("p2pFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *P2P) setFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
	cmd.MarkFlagFilename("p2p-config-file")
	cmd.MarkFlagDirname("p2p-discovery-dir")
}

func (c *P2P) mergeFlags() {
	if c.fs.Changed("p2p") {
		c.Enabled = true
	}
	if c.fs.Changed("p2p-only") {
		c.Only = true
	}
	if c.fs.Changed("p2p-webrtc-config-file") {
		c.WebRTCConfigurationFile, _ = c.fs.GetString("p2p-webrtc-config-file")
	}
	if c.fs.Changed("p2p-discovery-dir") {
		c.DiscoveryDir, _ = c.fs.GetString("p2p-discovery-dir")
	}
}

func (c *P2P) validate() error {
	// config file must be set if discovery dir is set
	if c.DiscoveryDir != "" && c.WebRTCConfigurationFile == "" {
		return errors.New("p2p-webrtc-config-file must be set if p2p-discovery-dir is set")
	}
	return nil
}

func (c *P2P) hydrate() error {
	if c.WebRTCConfigurationFile == "" {
		return nil
	}
	if c.WebRTCConfiguration != nil {
		return nil
	}

	data, err := os.ReadFile(c.WebRTCConfigurationFile)
	if err != nil {
		return fmt.Errorf("failed to read WebRTC configuration from file: %w", err)
	}

	var wv webrtc.Configuration
	if err = yaml.Unmarshal(data, &wv); err != nil {
		return fmt.Errorf("failed to unmarshal WebRTC configuration: %w", err)
	}

	c.WebRTCConfiguration = &wv

	return nil
}

type UPnP struct {
	ExternalPort int           `mapstructure:"externalPort" yaml:"externalPort"`
	Enabled      bool          `mapstructure:"mapPort" yaml:"mapPort"`
	Duration     time.Duration `mapstructure:"duration" yaml:"duration"`
	Timeout      time.Duration `mapstructure:"timeout" yaml:"timeout" flag:"upnp-discovery-timeout"`

	fs *pflag.FlagSet
}

func (c *UPnP) init() {
	c.fs = pflag.NewFlagSet("UPnP-IGD Flags", pflag.ExitOnError)

	c.fs.Int("external-port", 0, "External port to use for UPnP IGD port mapping.")
	c.fs.Duration("port-mapping-duration", 0, "Duration to use for UPnP IGD port mapping.")
	c.fs.Duration("upnp-discovery-timeout", 60*time.Second, "Timeout for UPnP IGD discovery.")
	c.fs.Bool("map-port", false, "Map port using UPnP IGD.")

	cobra.AddTemplateFunc("upnpFlags", func() *pflag.FlagSet {
		return c.fs
	})
}

func (c *UPnP) setFlags(cmd *cobra.Command, fs *pflag.FlagSet) {
	fs.AddFlagSet(c.fs)
	cmd.MarkFlagsRequiredTogether("external-port", "port-mapping-duration")
	cmd.MarkFlagsRequiredTogether("map-port", "port-mapping-duration")
}

func (c *UPnP) mergeFlags() {
	if c.fs.Changed("external-port") {
		c.ExternalPort, _ = c.fs.GetInt("external-port")
	}

	if c.fs.Changed("port-mapping-duration") {
		c.Duration, _ = c.fs.GetDuration("port-mapping-duration")
	}

	if c.fs.Changed("upnp-discovery-timeout") {
		c.Timeout, _ = c.fs.GetDuration("upnp-discovery-timeout")
	}

	if c.fs.Changed("map-port") {
		c.Enabled, _ = c.fs.GetBool("map-port")
	}
}

func (c *UPnP) validate() error {
	if c.ExternalPort < 0 || c.ExternalPort > 65535 {
		return errors.New("invalid external port")
	}

	if c.Duration < 0 {
		return errors.New("invalid port mapping duration")
	}

	if 0 < c.ExternalPort && c.Duration == 0 {
		return errors.New("port mapping duration must be specified when external port is specified")
	}

	if 0 < c.Duration && c.ExternalPort == 0 {
		return errors.New("external port must be specified when port mapping duration is specified")
	}

	if c.Enabled && c.Duration == 0 {
		return errors.New("port mapping duration must be specified when UPnP is enabled")
	}

	return nil
}
