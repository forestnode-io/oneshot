package redirect

import (
	"errors"
	"io"
	"net/http"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

func New(config *Configuration) *Cmd {
	return &Cmd{
		config: config,
	}
}

type Cmd struct {
	cobraCommand *cobra.Command
	config       *Configuration
	url          string
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "redirect url",
		Short: "Redirect all requests to the specified url",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			c.config.MergeFlags()
			return c.config.Validate()
		},
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

	c.config.SetFlags(c.cobraCommand, c.cobraCommand.Flags())

	return c.cobraCommand
}

func (c *Cmd) setHandlerFunc(cmd *cobra.Command, args []string) error {
	var ctx = cmd.Context()

	output.IncludeBody(ctx)

	c.url = args[0]

	commands.SetHTTPHandlerFunc(ctx, c.ServeHTTP)
	return nil
}

func (c *Cmd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := c.cobraCommand.Context()
	doneReadingBody := make(chan struct{})
	events.Raise(ctx, output.NewHTTPRequest(r))

	var header = http.Header(c.config.Header)
	for key := range header {
		w.Header().Set(key, header.Get(key))
	}

	go func() {
		defer close(doneReadingBody)
		defer r.Body.Close()
		_, _ = io.Copy(io.Discard, r.Body)
	}()

	http.Redirect(w, r, c.url, c.config.StatusCode)

	events.Success(ctx)
	<-doneReadingBody
}
