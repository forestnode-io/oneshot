package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/raphaelreyna/oneshot/v2/internal/events"
	oneshotnet "github.com/raphaelreyna/oneshot/v2/internal/net"
)

var ErrTimeout = errors.New("timeout")

type Server struct {
	HandlerFunc       http.HandlerFunc
	TLSCert, TLSKey   string
	Middleware        []Middleware
	BufferRequestBody bool
	Timeout           time.Duration

	http.Server
}

func (s *Server) Serve(ctx context.Context, queueSize int64, l net.Listener) error {
	// demux the handler and apply middleware
	httpHandler, cancelDemux := demux(queueSize, s.HandlerFunc)
	defer cancelDemux()
	for _, mw := range s.Middleware {
		httpHandler = mw(httpHandler)
	}

	// create the router and register the demuxed handler
	r := mux.NewRouter()
	r.PathPrefix("/").HandlerFunc(httpHandler)
	s.Handler = r

	if 0 < s.Timeout {
		l = oneshotnet.NewListenerTimer(l, s.Timeout)
	}

	go func() {
		<-ctx.Done()
		s.Server.Close()
	}()

	var err error
	if s.TLSKey != "" {
		err = s.Server.ServeTLS(l, s.TLSCert, s.TLSKey)
	} else {
		err = s.Server.Serve(l)
	}
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			if err := events.GetCancellationError(ctx); err != nil {
				// TODO(raphaelreyna) improve this
				panic(err)
			}
		}
	}

	return nil
}
