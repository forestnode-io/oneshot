package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
)

type testHandler struct {
	serveHTTP        func(http.ResponseWriter, *http.Request) (interface{}, error)
	serveExpiredHTTP func(http.ResponseWriter, *http.Request)
}

func (th *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return th.serveHTTP(w, r)
}

func (th *testHandler) ServeExpiredHTTP(w http.ResponseWriter, r *http.Request) {
	th.serveExpiredHTTP(w, r)
}

func TestNewServer(t *testing.T) {
	var (
		is = is.New(t)

		th testHandler
	)

	s := NewServer(&th)

	is.True(s != nil)
	is.True(s.requestsQueue != nil)
	is.True(s.handler == &th)
	is.True(s.successfulResult == nil)
}

func TestServer_Serve(t *testing.T) {
	var (
		is = is.New(t)

		ctx        = context.Background()
		reqCounter = 0
		th         = testHandler{
			serveHTTP: func(w http.ResponseWriter, r *http.Request) (interface{}, error) {
				if reqCounter < 1 {
					reqCounter++
					w.WriteHeader(http.StatusTeapot)
					payload := "NOT OK"
					w.Write([]byte(payload))
					return "not yet", errors.New("ERROR")
				}

				payload := "OK"
				summary := struct{}{}

				w.Write([]byte(payload))

				return summary, nil
			},
			serveExpiredHTTP: func(w http.ResponseWriter, r *http.Request) {
				status := http.StatusGone
				http.Error(w, http.StatusText(status), status)
			},
		}
	)

	s := NewServer(&th)
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

		is.True(s.results[0].(string) == "not yet")
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

		is.True(s.successfulResult == struct{}{})
	})

	t.Run("EOF", func(t *testing.T) {
		is := is.New(t)

		resp, err := http.Get("http://127.0.0.1:8080")
		is.True(errors.Is(err, io.EOF))
		is.True(resp == nil)
	})
}

func TestServer_Serve_Expired(t *testing.T) {
	var (
		is = is.New(t)

		ctx = context.Background()
		wg  = sync.WaitGroup{}
		th  = testHandler{
			serveHTTP: func(w http.ResponseWriter, r *http.Request) (interface{}, error) {
				wg.Wait()

				payload := "OK"
				summary := struct{}{}

				w.Write([]byte(payload))

				return summary, nil
			},
			serveExpiredHTTP: func(w http.ResponseWriter, r *http.Request) {
				status := http.StatusGone
				http.Error(w, http.StatusText(status), status)
			},
		}
	)

	s := NewServer(&th)
	l, err := net.Listen("tcp", ":8080")
	is.NoErr(err)

	defer l.Close()

	go s.Serve(ctx, l)

	wg.Add(1)

	// first request that blocks until second request is made
	go func() {
		is := is.New(t)
		resp, err := http.Get("http://127.0.0.1:8080")
		is.NoErr(err)
		is.True(resp.StatusCode == http.StatusOK)
	}()

	// second request that gets the expired content
	go func() {
		is := is.New(t)
		resp, err := http.Get("http://127.0.0.1:8080")
		is.NoErr(err)
		is.True(resp.StatusCode == http.StatusOK)
	}()
	time.Sleep(time.Second)
	wg.Done()
}
