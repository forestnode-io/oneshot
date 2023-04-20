package webrtc

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"io"
	"os"

	"github.com/pion/datachannel"
	"github.com/pion/webrtc/v3"
	"github.com/oneshot-uno/oneshot/v2/pkg/output"
)

const DataChannelName = "oneshot"

const (
	DataChannelMTU             = 16384              // 16 KB
	BufferedAmountLowThreshold = 1 * DataChannelMTU // 2^0 MTU
	MaxBufferedAmount          = 8 * DataChannelMTU // 2^3 MTUs
)

type DataChannelByteReader struct {
	datachannel.ReadWriteCloser
}

func (r *DataChannelByteReader) Read(p []byte) (int, error) {
	n, isString, err := r.ReadWriteCloser.ReadDataChannel(p)
	if err == nil && isString {
		err = io.EOF
	}
	return n, err
}

type ICEServer struct {
	URLs                  []string `yaml:"urls" mapstructure:"urls"`
	Username              string   `yaml:"username" mapstructure:"username"`
	Credential            []byte   `yaml:"credential" mapstructure:"credential"`
	CredentialPath        string   `yaml:"credentialPath" mapstructure:"credentialPath"`
	CredentialTypeIsOAuth bool     `yaml:"credentialTypeIsOAuth" mapstructure:"credentialTypeIsOAuth"`
}

type Certificate struct {
	PrivateKey     []byte `yaml:"privateKey" mapstructure:"privateKey"`
	PrivateKeyPath string `yaml:"privateKeyPath" mapstructure:"privateKeyPath"`
	Type           string `yaml:"type" mapstructure:"type"` //rsa or ecdsa
}

type Configuration struct {
	ICEServers   []*ICEServer   `yaml:"iceServers" mapstructure:"iceServers"`
	RelayOnly    bool           `yaml:"relayOnly" mapstructure:"relayOnly"`
	Certificates []*Certificate `yaml:"certificates" mapstructure:"certificates"`
}

func (c *Configuration) WebRTCConfiguration() (*webrtc.Configuration, error) {
	config := webrtc.Configuration{}

	if c.RelayOnly {
		config.ICETransportPolicy = webrtc.ICETransportPolicyRelay
	} else {
		config.ICETransportPolicy = webrtc.ICETransportPolicyAll
	}

	if len(c.ICEServers) == 0 {
		return nil, output.UsageErrorF("no ICE servers configured")
	}
	for _, s := range c.ICEServers {
		if len(s.URLs) == 0 {
			return nil, output.UsageErrorF("no URLs configured for ICE server")
		}
		if s.CredentialPath != "" {
			data, err := os.ReadFile(s.CredentialPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read credential path: %w", err)
			}
			s.Credential = data
		}

		config.ICEServers = append(config.ICEServers, webrtc.ICEServer{
			URLs:           s.URLs,
			Username:       s.Username,
			Credential:     s.Credential,
			CredentialType: webrtc.ICECredentialTypePassword,
		})
	}
	for _, cert := range c.Certificates {
		if cert.PrivateKeyPath == "" {
			return nil, output.UsageErrorF("no private key path configured for certificate")
		}

		if cert.PrivateKeyPath != "" {
			data, err := os.ReadFile(cert.PrivateKeyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read private key path: %w", err)
			}
			cert.PrivateKey = data
		}

		var (
			pk  crypto.PrivateKey
			err error
		)
		switch cert.Type {
		case "rsa":
			pk, err = x509.ParsePKCS1PrivateKey(cert.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}
		case "ecdsa":
			pk, err = x509.ParseECPrivateKey(cert.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}
		case "":
			pkIface, err := x509.ParsePKCS8PrivateKey(cert.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}
			pk = pkIface.(crypto.PrivateKey)
		default:
			return nil, fmt.Errorf("unknown private key type: %s", cert.Type)
		}

		cert, err := webrtc.GenerateCertificate(pk)
		if err != nil {
			return nil, fmt.Errorf("failed to generate certificate: %w", err)
		}

		config.Certificates = append(config.Certificates, *cert)
	}

	return &config, nil
}
