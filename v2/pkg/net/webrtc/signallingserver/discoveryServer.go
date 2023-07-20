package signallingserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/signallingserver/headers"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/signallingserver/messages"
	"github.com/oneshot-uno/oneshot/v2/pkg/net/webrtc/signallingserver/proto"
	"github.com/rs/zerolog"
	"golang.org/x/mod/semver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

var ErrClosedByUser = errors.New("closed by user")

var kacp = keepalive.ClientParameters{
	Time:    6 * time.Second, // send pings every 6 seconds if there is no activity
	Timeout: time.Second,     // wait 1 second for ping ack before considering the connection dead
}
var dialTimeout = 3 * time.Second

type discoveryServerKey struct{}

type DiscoveryServer struct {
	conn        *grpc.ClientConn
	stream      proto.SignallingServer_ConnectClient
	AssignedURL string
	config      *DiscoveryServerConfig
	arrival     *messages.ServerArrivalRequest
}

func (d *DiscoveryServer) Stream() proto.SignallingServer_ConnectClient {
	return d.stream
}

type DiscoveryServerConfig struct {
	URL      string
	Key      string
	Insecure bool
	TLSCert  string
	TLSKey   string

	VersionInfo messages.VersionInfo
}

// WithDiscoveryServer connects to the discovery server and sends a handshake and arrival.
// The connnection is kept open and stuffed into the context.
func WithDiscoveryServer(ctx context.Context, c DiscoveryServerConfig, arrival messages.ServerArrivalRequest) (context.Context, error) {
	log := zerolog.Ctx(ctx)

	log.Debug().Msg("connecting to discovery server")
	conn, err := getConnectionToDiscoveryServer(ctx, &c)
	if err != nil {
		return ctx, fmt.Errorf("failed to connect to discovery server: %w", err)
	}

	log.Debug().Msg("opening bidirectional stream to discovery server")
	ds, err := newDiscoveryServer(ctx, &c, &arrival, conn)
	if err != nil {
		return ctx, fmt.Errorf("failed to create discovery server: %w", err)
	}

	log.Debug().Msg("negotiating arrival with discovery server")
	if err := ds.negotiateArrival(ctx, &arrival); err != nil {
		return ctx, fmt.Errorf("failed to negotiate arrival with discovery server: %w", err)
	}

	return context.WithValue(ctx, discoveryServerKey{}, ds), nil
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
		// check if the discovery server closed the connection by user request
		if errors.Is(err, io.EOF) {
			trailer := d.stream.Trailer()
			values := trailer.Get(headers.ClosedByUser)
			if len(values) > 0 {
				v := values[0]
				if v == "true" {
					return nil, ErrClosedByUser
				}
			}
		}
		return nil, fmt.Errorf("failed to receive message from discovery server: %w", err)
	}
	m, err := messages.FromRPCEnvelope(env)
	if err != nil {
		return nil, fmt.Errorf("failed to convert RPC envelope to message: %w", err)
	}
	return m, nil
}

func (d *DiscoveryServer) negotiateArrival(ctx context.Context, arrival *messages.ServerArrivalRequest) error {
	var (
		log  = zerolog.Ctx(ctx)
		conf = d.config

		closeConn = true
	)
	defer func() {
		if closeConn {
			d.Close()
		}
	}()

	log.Debug().
		Msg("sending handshake to discovery server")
	err := Send(d, &messages.Handshake{
		ID:          conf.Key,
		VersionInfo: conf.VersionInfo,
	})
	if err != nil {
		return fmt.Errorf("failed to send handshake to discovery server: %w", err)
	}

	log.Debug().
		Msg("waiting for discovery server to respond to handshake")

	// Receive the handshake response with a timeout.
	// We probably only really need a timeout on the first message.

	type hserr struct {
		err error
		hs  *messages.Handshake
	}
	hserrChan := make(chan hserr, 1)
	go func() {
		hs, err := Receive[*messages.Handshake](d)
		hserrChan <- hserr{err: err, hs: hs}
		close(hserrChan)
	}()

	var hs *messages.Handshake
	select {
	case <-time.After(1 * time.Second):
		return ErrHandshakeTimeout
	case <-ctx.Done():
		return ctx.Err()
	case hserr := <-hserrChan:
		if hserr.err != nil {
			return fmt.Errorf("failed to receive handshake from discovery server: %w", hserr.err)
		}
		if hserr.hs.Error != "" {
			return fmt.Errorf("discovery server returned error: %s", hserr.hs.Error)
		}
		hs = hserr.hs
	}

	log.Debug().
		Str("version", hs.VersionInfo.Version).
		Str("api-version: ", hs.VersionInfo.APIVersion).
		Msg("discovery server handshake successful")

	// Check if the discovery server is running a newer version of the API than this client.
	// The discovery server should be backwards compatible with older clients.
	if semver.Compare(hs.VersionInfo.APIVersion, conf.VersionInfo.APIVersion) < 0 {
		return fmt.Errorf("discovery server is running an older version of the API (%s) than this client (%s)", hs.VersionInfo.APIVersion, conf.VersionInfo.APIVersion)
	}

	log.Debug().
		Interface("arrival-request", arrival).
		Msg("sending server arrival request")

	if err = Send(d, arrival); err != nil {
		return fmt.Errorf("failed to send server arrival request to discovery server: %w", err)
	}

	// Wait for the discovery server to acknowledge the arrival.
	log.Debug().Msg("waiting for discovery server to acknowledge arrival")
	sar, err := Receive[*messages.ServerArrivalResponse](d)
	if err != nil {
		return fmt.Errorf("failed to receive server arrival response from discovery server: %w", err)
	}
	if sar.Error != "" {
		return fmt.Errorf("discovery server returned error: %s", sar.Error)
	}
	if sar.AssignedURL == "" {
		return fmt.Errorf("discovery server did not assign a URL")
	}

	closeConn = false
	d.AssignedURL = sar.AssignedURL
	log.Debug().Msg("discovery server acknowledged arrival")
	log.Info().
		Str("assigned-url", d.AssignedURL).
		Msg("negotiated arrival with discovery server")

	return nil
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

func ReconnectDiscoveryServer(ctx context.Context) error {
	ds := GetDiscoveryServer(ctx)
	if ds == nil {
		return nil
	}
	ds.conn.Close()

	log := zerolog.Ctx(ctx)
	config := ds.config
	arrival := ds.arrival
	arrival.PreviouslyAssignedURL = ds.AssignedURL

	log.Debug().Msg("connecting to discovery server")
	conn, err := getConnectionToDiscoveryServer(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to connect to discovery server: %w", err)
	}

	log.Debug().Msg("opening bidirectional stream to discovery server")
	newDS, err := newDiscoveryServer(ctx, config, arrival, conn)
	if err != nil {
		return fmt.Errorf("failed to create discovery server: %w", err)
	}

	log.Debug().Msg("negotiating arrival with discovery server")
	if err := newDS.negotiateArrival(ctx, arrival); err != nil {
		return fmt.Errorf("failed to negotiate arrival with discovery server: %w", err)
	}

	*ds = *newDS

	return nil
}

func getConnectionToDiscoveryServer(ctx context.Context, config *DiscoveryServerConfig) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(kacp),
		grpc.FailOnNonTempDialError(true),
	}

	if config.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		if config.TLSCert != "" {
			if config.TLSKey == "" {
				return nil, fmt.Errorf("missing TLS key")
			}
		}
		if config.TLSKey != "" {
			if config.TLSCert == "" {
				return nil, fmt.Errorf("missing TLS cert")
			}
		}

		var tlsConf *tls.Config
		if config.TLSCert != "" && config.TLSKey != "" {
			cert, err := tls.LoadX509KeyPair(config.TLSCert, config.TLSKey)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS keypair: %w", err)
			}
			tlsConf = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConf)))
	}

	dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	return grpc.DialContext(dialCtx, config.URL, opts...)
}

func newDiscoveryServer(ctx context.Context, config *DiscoveryServerConfig, arrival *messages.ServerArrivalRequest, conn *grpc.ClientConn) (*DiscoveryServer, error) {
	stream, err := proto.NewSignallingServerClient(conn).Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to discovery server: %w", err)
	}

	return &DiscoveryServer{
		conn:    conn,
		stream:  stream,
		config:  config,
		arrival: arrival,
	}, nil
}

var ErrHandshakeTimeout = errors.New("handshake timed out")
