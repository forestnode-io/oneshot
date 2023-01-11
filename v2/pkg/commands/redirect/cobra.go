package redirect

import (
	"errors"
	"net/http"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

func New() *Cmd {
	return &Cmd{
		header: make(http.Header),
	}
}

type Cmd struct {
	cobraCommand *cobra.Command
	header       http.Header
	statusCode   int
	url          string
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.header == nil {
		c.header = make(http.Header)
	}
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:  "redirect url",
		RunE: c.setHandlerFunc,
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

	flags := c.cobraCommand.LocalFlags()
	flags.IntP("status-code", "s", http.StatusTemporaryRedirect, "HTTP status code")

	return c.cobraCommand
}

func (c *Cmd) setHandlerFunc(cmd *cobra.Command, args []string) error {
	var (
		ctx = cmd.Context()

		flags                = c.cobraCommand.Flags()
		statCode, statCodeOk = flags.GetInt("status-code")
		headerSlice, _       = flags.GetStringSlice("header")
	)

	if statCodeOk != nil {
		statCode = http.StatusTemporaryRedirect
	}

	c.url = args[0]
	c.statusCode = statCode
	c.header = oneshothttp.HeaderFromStringSlice(headerSlice)

	commands.SetHTTPHandlerFunc(ctx, c.ServeHTTP)
	return nil
}

func (c *Cmd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := c.cobraCommand.Context()
	output.Raise(ctx, output.NewHTTPRequest(r))

	var header = c.header
	for key := range header {
		w.Header().Set(key, header.Get(key))
	}

	http.Redirect(w, r, c.url, c.statusCode)

	events.Success(ctx)
}
