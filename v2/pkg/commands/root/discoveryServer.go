package root

import (
	"context"
	"crypto/sha256"
	"fmt"

	oneshotnet "github.com/oneshot-uno/oneshot/v2/pkg/net"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/signallingserver"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/oneshot-uno/oneshot/v2/pkg/version"
	"golang.org/x/crypto/bcrypt"
)

func (r *rootCommand) withDiscoveryServer(ctx context.Context) (context.Context, error) {
	var (
		config   = r.config
		dsConfig = config.NATTraversal.DiscoveryServer
	)

	if dsConfig.URL == "" {
		return ctx, nil
	}

	var (
		connConf = signallingserver.DiscoveryServerConfig{
			URL:      dsConfig.URL,
			Key:      dsConfig.Key,
			Insecure: dsConfig.Insecure,
			VersionInfo: messages.VersionInfo{
				Version:    version.Version,
				APIVersion: version.APIVersion,
			},
		}
		arrival = messages.ServerArrivalRequest{
			IsUsingPortMapping: config.NATTraversal.IsUsingUPnP(),
			RedirectOnly:       !config.NATTraversal.P2P.Enabled,
			TTL:                config.Server.Timeout,
		}
	)

	ipThatCanReachDiscoveryServer, err := oneshotnet.GetSourceIP(dsConfig.URL, 80)
	if err != nil {
		return ctx, fmt.Errorf("unable to reach the discovery server: %w", err)
	}
	arrival.Redirect = ipThatCanReachDiscoveryServer

	scheme := "http"
	if config.Server.TLSCert != "" {
		scheme = "https"
	}
	arrival.Redirect = fmt.Sprintf("%s://%s:%d", scheme, arrival.Redirect, config.Server.Port)

	if arrival.IsUsingPortMapping {
		if d := config.NATTraversal.UPnP.Duration; d != 0 && d < arrival.TTL {
			arrival.TTL = d
		}
	}

	if dsConfig.PreferredURL != "" || dsConfig.RequiredURL != "" {
		switch {
		case dsConfig.RequiredURL != "":
			arrival.URL = &messages.SessionURLRequest{
				URL:      dsConfig.RequiredURL,
				Required: true,
			}
		case dsConfig.PreferredURL != "":
			arrival.URL = &messages.SessionURLRequest{
				URL:      dsConfig.PreferredURL,
				Required: false,
			}
		}
	}

	if !arrival.RedirectOnly {
		var (
			baConf   = config.BasicAuth
			username = baConf.Username
			password = baConf.Password
			bam      *messages.BasicAuth
		)

		if username != "" || password != "" {
			bam = &messages.BasicAuth{}
			if username != "" {
				uHash := sha256.Sum256([]byte(username))
				bam.UsernameHash = uHash[:]
			}
			if password != "" {
				pHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if err != nil {
					return ctx, fmt.Errorf("failed to hash password: %w", err)
				}
				bam.PasswordHash = pHash
			}
		}

		arrival.BasicAuth = bam
	}

	return signallingserver.WithDiscoveryServer(ctx, connConf, arrival)
}
