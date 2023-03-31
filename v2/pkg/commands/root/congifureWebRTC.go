package root

import (
	"fmt"
	"os"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func (r *rootCommand) configureWebRTC(flags *pflag.FlagSet) error {
	path, _ := flags.GetString("p2p-config-file")
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read p2p config file: %w", err)
	}

	config := webrtc.Configuration{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("unable to parse p2p config file: %w", err)
	}

	r.webrtcConfig, err = config.WebRTCConfiguration()
	if err != nil {
		return fmt.Errorf("unable to configure p2p: %w", err)
	}

	return nil
}
