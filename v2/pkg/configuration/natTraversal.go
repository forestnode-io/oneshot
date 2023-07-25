package configuration

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/forestnode-io/oneshot/v2/pkg/flags"
	"github.com/forestnode-io/oneshot/v2/pkg/net/webrtc"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type NATTraversal struct {
	P2P  P2P  `mapstructure:"p2p" yaml:"p2p"`
	UPnP UPnP `mapstructure:"upnp" yaml:"upnp"`
}

func (c *NATTraversal) IsUsingWebRTC() bool {
	return c.P2P.Enabled || c.P2P.Only
}

func (c *NATTraversal) IsUsingUPnP() bool {
	return c.UPnP.Enabled || c.UPnP.ExternalPort > 0 || c.UPnP.Duration > 0
}

func setNATTraversalFlags(cmd *cobra.Command) {
	setP2PFlags(cmd)
	setUPnPFlags(cmd)
}

func (c *NATTraversal) validate() error {
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

type P2P struct {
	Enabled                 bool          `mapstructure:"enabled" yaml:"enabled"`
	Only                    bool          `mapstructure:"only" yaml:"only"`
	WebRTCConfigurationFile string        `mapstructure:"webrtcConfigurationFile" yaml:"webrtcConfigurationFile"`
	WebRTCConfiguration     []byte        `json:"webrtcConfiguration" yaml:"webrtcConfiguration"`
	DiscoveryDir            string        `mapstructure:"discoveryDir" yaml:"discoveryDir"`
	ICEGatherTimeout        time.Duration `mapstructure:"iceGatherTimeout" yaml:"iceGatherTimeout"`
}

func setP2PFlags(cmd *cobra.Command) {
	fs := pflag.NewFlagSet("P2P", pflag.ExitOnError)
	defer cmd.PersistentFlags().AddFlagSet(fs)

	flags.Bool(fs, "nattraversal.p2p.enabled", "p2p", "Accept incoming p2p connections. Requires a discovery mechanism, either a discovery server or a discovery directory.")
	flags.Bool(fs, "nattraversal.p2p.only", "p2p-only", "Only accept incoming p2p connections.")
	flags.String(fs, "nattraversal.p2p.webrtcconfigurationfile", "p2p-webrtc-config-file", "Path to the configuration file for the underlying WebRTC transport.")
	flags.String(fs, "nattraversal.p2p.discoverydir", "p2p-discovery-dir", "Path to the directory containing the discovery files. In this directory, each peer connection has a numerically named subdirectory containing an answer and offer file. The offer file contains the RTCSessionDescription JSON of the WebRTc offer and the answer file contains the RTCSessionDescription JSON of the WebRTC answer.")

	cobra.AddTemplateFunc("p2pFlags", func() *pflag.FlagSet {
		return fs
	})
	cobra.AddTemplateFunc("p2pClientFlags", func() *pflag.FlagSet {
		fs := pflag.NewFlagSet("P2P Client", pflag.ExitOnError)
		fs.String("p2p-discovery-dir", "", `Path to the directory containing the discovery files.
In this directory, each peer connection has a numerically named subdirectory containing an answer and offer file.
The offer file contains the RTCSessionDescription JSON of the WebRTc offer
and the answer file contains the RTCSessionDescription JSON of the WebRTC answer.`)

		return fs
	})
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

	c.WebRTCConfiguration = data

	return nil
}

func (c *P2P) ParseConfig() (*webrtc.Configuration, error) {
	if c.WebRTCConfiguration == nil {
		return nil, nil
	}

	var wv webrtc.Configuration
	if err := yaml.Unmarshal(c.WebRTCConfiguration, &wv); err != nil {
		return nil, fmt.Errorf("failed to unmarshal WebRTC configuration: %w", err)
	}

	return &wv, nil
}

type UPnP struct {
	ExternalPort int           `mapstructure:"externalPort" yaml:"externalPort"`
	Enabled      bool          `mapstructure:"mapPort" yaml:"mapPort"`
	Duration     time.Duration `mapstructure:"duration" yaml:"duration"`
	Timeout      time.Duration `mapstructure:"timeout" yaml:"timeout" flag:"upnp-discovery-timeout"`
}

func setUPnPFlags(cmd *cobra.Command) {
	fs := pflag.NewFlagSet("UPnP-IGD Flags", pflag.ExitOnError)
	defer cmd.Flags().AddFlagSet(fs)

	flags.Int(fs, "nattraversal.upnp.externalport", "external-port", "External port to use for UPnP IGD port mapping.")
	flags.Duration(fs, "nattraversal.upnp.duration", "port-mapping-duration", "Duration to use for UPnP IGD port mapping.")
	flags.Duration(fs, "nattraversal.upnp.timeout", "upnp-discovery-timeout", "Timeout for UPnP IGD discovery.")
	flags.Bool(fs, "nattraversal.upnp.enabled", "map-port", "Map port using UPnP IGD.")

	cobra.AddTemplateFunc("upnpFlags", func() *pflag.FlagSet {
		return fs
	})
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

	if (c.Enabled || c.ExternalPort != 0) && c.Timeout == 0 {
		return errors.New("UPnP discovery timeout must be specified when UPnP is enabled or external port is specified")
	}

	return nil
}
