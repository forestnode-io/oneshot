package signallingserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/raphaelreyna/oneshot/v2/pkg/net/webrtc/signallingserver/proto"
	"golang.org/x/mod/semver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

var kacp = keepalive.ClientParameters{
	Time:    6 * time.Second, // send pings every 6 seconds if there is no activity
	Timeout: time.Second,     // wait 1 second for ping ack before considering the connection dead
}
var dialTimeout = 3 * time.Second

type discoveryServerKey struct{}

type DiscoveryServer struct {
	conn   *grpc.ClientConn
	stream proto.SignallingServer_ConnectClient
}

type DiscoveryServerConfig struct {
	URL      string
	Key      string
	Insecure bool
	TLSCert  string
	TLSKey   string

	VersionInfo messages.VersionInfo
}

func WithDiscoveryServer(ctx context.Context, c DiscoveryServerConfig) (context.Context, error) {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(kacp),
	}
	if c.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		if c.TLSCert != "" {
			if c.TLSKey == "" {
				return ctx, fmt.Errorf("missing TLS key")
			}
		}
		if c.TLSKey != "" {
			if c.TLSCert == "" {
				return ctx, fmt.Errorf("missing TLS cert")
			}
		}

		var tlsConf *tls.Config
		if c.TLSCert != "" && c.TLSKey != "" {
			cert, err := tls.LoadX509KeyPair(c.TLSCert, c.TLSKey)
			if err != nil {
				return ctx, fmt.Errorf("failed to load TLS keypair: %w", err)
			}
			tlsConf = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConf)))
	}

	dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, c.URL, opts...)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ctx, fmt.Errorf("timeout dialing discovery server")
		}
		if errors.Is(err, context.Canceled) {
			return ctx, fmt.Errorf("canceled dialing discovery server")
		}
		return ctx, fmt.Errorf("failed to dial discovery server: %w", err)
	}
	stream, err := proto.NewSignallingServerClient(conn).Connect(ctx)
	if err != nil {
		return ctx, fmt.Errorf("failed to connect to discovery server: %w", err)
	}

	ds := DiscoveryServer{
		conn:   conn,
		stream: stream,
	}

	err = Send(&ds, &messages.Handshake{
		ID:          c.Key,
		VersionInfo: c.VersionInfo,
	})
	if err != nil {
		return ctx, fmt.Errorf("failed to send handshake to discovery server: %w", err)
	}

	hs, err := Receive[*messages.Handshake](&ds)
	if err != nil {
		return ctx, fmt.Errorf("failed to receive handshake from discovery server: %w", err)
	}

	if hs.Error != "" {
		ds.Close()
		return ctx, fmt.Errorf("discovery server returned error: %s", hs.Error)
	}

	// Check if the discovery server is running a newer version of the API than this client.
	// The discovery server should be backwards compatible with older clients.
	if semver.Compare(hs.VersionInfo.APIVersion, c.VersionInfo.APIVersion) < 0 {
		ds.Close()
		return ctx, fmt.Errorf("discovery server is running an older version of the API (%s) than this client (%s)", hs.VersionInfo.Version, c.VersionInfo.Version)
	}

	return context.WithValue(ctx, discoveryServerKey{}, &ds), nil
}

func GetDiscoveryServer(ctx context.Context) *DiscoveryServer {
	ds, ok := ctx.Value(discoveryServerKey{}).(*DiscoveryServer)
	if !ok {
		return nil
	}
	return ds
}

func CloseDiscoveryServer(ctx context.Context) error {
	ds := GetDiscoveryServer(ctx)
	if ds == nil {
		return nil
	}
	return ds.Close()
}

func (d *DiscoveryServer) send(m messages.Message) error {
	env, err := messages.ToRPCEnvelope(m)
	if err != nil {
		return fmt.Errorf("failed to convert message to RPC envelope: %w", err)
	}
	return d.stream.Send(env)
}

func (d *DiscoveryServer) recv() (messages.Message, error) {
	env, err := d.stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive message from discovery server: %w", err)
	}
	m, err := messages.FromRPCEnvelope(env)
	if err != nil {
		return nil, fmt.Errorf("failed to convert RPC envelope to message: %w", err)
	}
	return m, nil
}

func (d *DiscoveryServer) Close() error {
	d.stream.CloseSend()
	return d.conn.Close()
}

func Send[M messages.Message](d *DiscoveryServer, m M) error {
	return d.send(m)
}

func Receive[M messages.Message](d *DiscoveryServer) (M, error) {
	var (
		m   M
		mi  messages.Message
		err error
	)

	mi, err = d.recv()
	if err != nil {
		return m, err
	}

	m, ok := mi.(M)
	if !ok {
		return m, fmt.Errorf("expected message of type %T but got %T", m, mi)
	}

	return m, nil
}
