package api

import (
	"net/http"

	"github.com/raphaelreyna/oneshot/internal/server"
	"github.com/raphaelreyna/oneshot/v2/internal/out"
	"github.com/spf13/cobra"
)

type HTTPHandler func(Context, http.ResponseWriter, *http.Request)

type Cmd interface {
	Cobra() *cobra.Command
	Server() *server.Server
}

type Context interface {
	Success()
	Raise(out.Event)
}
