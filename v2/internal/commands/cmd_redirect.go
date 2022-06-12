package commands

import (
	"errors"
	"net/http"

	"github.com/raphaelreyna/oneshot/v2/internal/server"
	"github.com/raphaelreyna/oneshot/v2/internal/summary"
	"github.com/spf13/cobra"
)

func init() {
	x := redirectCmd{
		header: make(http.Header),
	}
	root.AddCommand(x.command())
}

type redirectCmd struct {
	cobraCommand *cobra.Command
	header       http.Header
	statusCode   int
	url          string
}

func (c *redirectCmd) command() *cobra.Command {
	c.cobraCommand = &cobra.Command{
		Use:  "redirect url",
		RunE: c.runE,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("redirect url required")
			}
			if 1 < len(args) {
				return errors.New("too many arguments, only 1 url may be used")
			}
			return nil
		},
	}

	flags := c.cobraCommand.Flags()
	flags.IntP("status-code", "s", http.StatusTemporaryRedirect, "HTTP status code")

	return c.cobraCommand
}

func (c *redirectCmd) runE(cmd *cobra.Command, args []string) error {
	var (
		ctx = cmd.Context()

		flags          = cmd.Flags()
		statCode, _    = flags.GetInt("status-code")
		headerSlice, _ = flags.GetStringSlice("header")
	)

	c.url = args[0]
	c.statusCode = statCode
	c.header = headerFromStringSlice(headerSlice)

	srvr := server.NewServer(c)
	setServer(ctx, srvr)
	return nil
}

func (c *redirectCmd) ServeHTTP(w http.ResponseWriter, r *http.Request) (*summary.Request, error) {
	var (
		sr     = summary.NewRequest(r)
		header = c.header
	)

	for key := range header {
		w.Header().Set(key, header.Get(key))
	}

	http.Redirect(w, r, c.url, c.statusCode)
	return sr, nil
}

func (s *redirectCmd) ServeExpiredHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("expired hello from server"))
}
