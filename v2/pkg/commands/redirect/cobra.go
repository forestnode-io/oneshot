package redirect

import (
	"fmt"
	"io"
	"net/http"

	"github.com/oneshot-uno/oneshot/v2/pkg/commands"
	"github.com/oneshot-uno/oneshot/v2/pkg/configuration"
	"github.com/oneshot-uno/oneshot/v2/pkg/events"
	"github.com/oneshot-uno/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

func New(config *configuration.Root) *Cmd {
	return &Cmd{
		config: config,
	}
}

type Cmd struct {
	cobraCommand *cobra.Command
	config       *configuration.Root
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
			config := c.config.Subcommands.Redirect
			config.MergeFlags()
			if err := config.Validate(); err != nil {
				return output.UsageErrorF("invalid configuration: %w", err)
			}
			if err := config.Hydrate(); err != nil {
				return fmt.Errorf("failed to hydrate configuration: %w", err)
			}
			return nil
		},
		RunE: c.setHandlerFunc,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return output.UsageErrorF("redirect url required")
			}
			if 1 < len(args) {
				return output.UsageErrorF("too many arguments, only 1 url may be used")
			}
			return nil
		},
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)

	c.config.Subcommands.Redirect.SetFlags(c.cobraCommand, c.cobraCommand.Flags())

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
	var (
		ctx    = c.cobraCommand.Context()
		config = c.config.Subcommands.Redirect
	)

	doneReadingBody := make(chan struct{})
	events.Raise(ctx, output.NewHTTPRequest(r))

	var header = http.Header(config.Header)
	for key := range header {
		w.Header().Set(key, header.Get(key))
	}

	go func() {
		defer close(doneReadingBody)
		defer r.Body.Close()
		_, _ = io.Copy(io.Discard, r.Body)
	}()

	http.Redirect(w, r, c.url, config.StatusCode)

	events.Success(ctx)
	<-doneReadingBody
}
