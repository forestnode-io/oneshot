package configuration

import (
	"fmt"
	"os"

	"github.com/oneshot-uno/oneshot/v2/pkg/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Discovery struct {
	Host         string `mapstructure:"host" yaml:"host"`
	Key          string `mapstructure:"key" yaml:"key" json:"-"`
	KeyPath      string `mapstructure:"keyPath" yaml:"keyPath"`
	Insecure     bool   `mapstructure:"insecure" yaml:"insecure"`
	PreferredURL string `mapstructure:"preferredURL" yaml:"preferredURL"`
	RequiredURL  string `mapstructure:"requiredURL" yaml:"requiredURL"`
	OnlyRedirect bool   `mapstructure:"onlyRedirect" yaml:"onlyRedirect"`
}

func setDiscoveryFlags(cmd *cobra.Command) {
	fs := pflag.NewFlagSet("Discovery Flags", pflag.ExitOnError)
	defer cmd.PersistentFlags().AddFlagSet(fs)

	flags.String(fs, "discovery.url", "discovery-url", "URL of the discovery server to connect to.")
	flags.String(fs, "discovery.keypath", "discovery-key-path", "Path to the key to present to the discovery server.")
	flags.String(fs, "discovery.key", "discovery-key", "Key to present to the discovery server.")
	fs.Lookup("discovery-key").DefValue = ""
	flags.Bool(fs, "discovery.insecure", "discovery-insecure", "Allow insecure connections to the discovery server.")
	flags.String(fs, "discovery.preferredurl", "discovery-preferred-url", "URL that the discovery server should try to reserve for connecting client.")
	flags.String(fs, "discovery.requiredurl", "discovery-required-url", "URL that the discovery server must reserve for connecting client.")
	flags.Bool(fs, "discovery.onlyredirect", "discovery-only-redirect", "Only redirect to this oneshot, do not use p2p.")

	cobra.AddTemplateFunc("discoveryFlags", func() *pflag.FlagSet {
		return fs
	})
}

func (c *Discovery) hydrate() error {
	if c.KeyPath != "" {
		key, err := os.ReadFile(c.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to read discovery server key file: %w", err)
		}
		c.Key = string(key)
	}

	return nil
}
