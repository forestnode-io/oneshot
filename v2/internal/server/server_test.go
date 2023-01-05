package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/matryer/is"
	"github.com/raphaelreyna/oneshot/v2/internal/api"
	"github.com/raphaelreyna/oneshot/v2/internal/out/events"
)

type testHandler struct {
	serveHTTP        api.HTTPHandler
	serveExpiredHTTP api.HTTPHandler
}

func (th *testHandler) ServeHTTP(actx api.Context, w http.ResponseWriter, r *http.Request) {
	th.serveHTTP(actx, w, r)
}

func (th *testHandler) ServeExpiredHTTP(actx api.Context, w http.ResponseWriter, r *http.Request) {
	th.serveExpiredHTTP(actx, w, r)
}

func TestNewServer(t *testing.T) {
	var (
		is = is.New(t)

		th testHandler
	)

	s := NewServer(th.ServeHTTP, th.ServeExpiredHTTP)

	is.True(s != nil)
	is.True(s.requestsQueue != nil)
	is.True(s.serveHTTP != nil)
	is.True(s.serveExpiredHTTP != nil)
	is.True(s.success == false)
}

func TestServer_Serve(t *testing.T) {
	var (
		is = is.New(t)

		ctx        = context.Background()
		reqCounter = 0
		th         = testHandler{
			serveHTTP: func(actx api.Context, w http.ResponseWriter, r *http.Request) {
				if reqCounter < 1 {
					reqCounter++
					w.WriteHeader(http.StatusTeapot)
					payload := "NOT OK"
					w.Write([]byte(payload))
					actx.Raise(&events.ClientDisconnected{
						Err: errors.New("ERROR"),
					})
					return
				}

				payload := "OK"
				w.Write([]byte(payload))
				actx.Success()
			},
			serveExpiredHTTP: func(actx api.Context, w http.ResponseWriter, r *http.Request) {
				status := http.StatusGone
				http.Error(w, http.StatusText(status), status)
			},
		}
	)

	s := NewServer(th.ServeHTTP, th.ServeExpiredHTTP)
	l, err := net.Listen("tcp", ":8080")
	is.NoErr(err)

	defer l.Close()

	go s.Serve(ctx, l)

	t.Run("failed transfer", func(t *testing.T) {
		is := is.New(t)

		resp, err := http.Get("http://127.0.0.1:8080")
		is.NoErr(err)
		is.True(resp.StatusCode == http.StatusTeapot)
		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		is.NoErr(err)
		is.Equal(string(body), "NOT OK")

		is.True(s.success == false)
	})

	t.Run("succesful transfer", func(t *testing.T) {
		is := is.New(t)

		resp, err := http.Get("http://127.0.0.1:8080")
		is.NoErr(err)
		is.True(resp.StatusCode == http.StatusOK)
		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		is.NoErr(err)
		is.Equal(string(body), "OK")

		is.True(s.success)
	})

	t.Run("EOF", func(t *testing.T) {
		is := is.New(t)

		resp, err := http.Get("http://127.0.0.1:8080")
		is.True(err != nil)
		is.True(resp == nil)
	})
}

func TestServer_Serve_Expired(t *testing.T) {
	var (
		is = is.New(t)

		ctx = context.Background()
		wg  = sync.WaitGroup{}
		th  = testHandler{
			serveHTTP: func(actx api.Context, w http.ResponseWriter, r *http.Request) {
				wg.Wait()
				payload := "OK"
				w.Write([]byte(payload))
				actx.Success()
				return
			},
			serveExpiredHTTP: func(actx api.Context, w http.ResponseWriter, r *http.Request) {
				status := http.StatusGone
				http.Error(w, http.StatusText(status), status)
			},
		}
	)

	s := NewServer(th.ServeHTTP, th.ServeExpiredHTTP)
	l, err := net.Listen("tcp", ":8080")
	is.NoErr(err)

	defer l.Close()

	go s.Serve(ctx, l)

	wg.Add(1)

	// first request that blocks until second request is made
	wg.Add(1)
	go func() {
		defer wg.Done()
		is := is.New(t)
		resp, err := http.Get("http://127.0.0.1:8080")
		is.NoErr(err)
		is.True(resp.StatusCode == http.StatusOK)
	}()

	// second request that gets the expired content
	wg.Add(1)
	go func() {
		defer wg.Done()
		is := is.New(t)
		resp, err := http.Get("http://127.0.0.1:8080")
		is.NoErr(err)
		is.True(resp.StatusCode == http.StatusOK)
	}()
	wg.Done()
}
