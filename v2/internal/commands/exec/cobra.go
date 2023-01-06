package exec

import (
	"io"
	"net/http"
	"os"

	"github.com/raphaelreyna/oneshot/v2/internal/cgi"
	"github.com/raphaelreyna/oneshot/v2/internal/commands"
	"github.com/raphaelreyna/oneshot/v2/internal/events"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/internal/net/http"
	"github.com/raphaelreyna/oneshot/v2/internal/out"
	"github.com/spf13/cobra"
)

type Cmd struct {
	cobraCommand *cobra.Command
	handler      *cgi.Handler
	header       http.Header
}

func New() *Cmd {
	return &Cmd{
		header: make(http.Header),
	}
}

func (c *Cmd) Cobra() *cobra.Command {
	c.cobraCommand = &cobra.Command{
		Use:  "exec command",
		RunE: c.createServer,
	}

	flags := c.cobraCommand.LocalFlags()
	flags.Bool("enforce-cgi", false, "The exec must conform to the CGI.")
	flags.StringSliceP("env", "e", []string{}, "Set an environment variable")
	flags.String("dir", "", "Set the working directory")
	flags.String("stderr", "", "Where to send exec stderr")
	flags.Bool("replace-headers", false, "Allow exec to replace header values")

	return c.cobraCommand
}

func (c *Cmd) createServer(cmd *cobra.Command, args []string) error {
	var (
		ctx = cmd.Context()

		flags          = cmd.Flags()
		headerSlice, _ = flags.GetStringSlice("header")
		stderr, _      = flags.GetString("stderr")

		strictCGI, _      = flags.GetBool("enforce-cgi")
		env, _            = flags.GetStringSlice("env")
		dir, _            = flags.GetString("dir")
		replaceHeaders, _ = flags.GetBool("replace-headers")
	)

	handlerConf := cgi.HandlerConfig{
		Cmd:           args,
		WorkingDir:    dir,
		InheritEnvs:   nil,
		BaseEnv:       env,
		Header:        oneshothttp.HeaderFromStringSlice(headerSlice),
		OutputHandler: cgi.DefaultOutputHandler,
		Stderr:        cmd.ErrOrStderr(),
	}

	switch {
	case replaceHeaders:
		handlerConf.OutputHandler = cgi.OutputHandlerReplacer
	case strictCGI:
		handlerConf.OutputHandler = cgi.DefaultOutputHandler
	default:
		handlerConf.OutputHandler = cgi.EZOutputHandler
	}
	if stderr != "" {
		var err error
		handlerConf.Stderr, err = os.Open(stderr)
		if err != nil {
			return err
		}
		commands.MarkForClose(ctx, handlerConf.Stderr.(io.WriteCloser))
	} else {
		handlerConf.Stderr = cmd.ErrOrStderr()
	}

	handler, err := cgi.NewHandler(handlerConf)
	if err != nil {
		return err
	}

	c.handler = handler
	commands.SetHTTPHandlerFunc(ctx, c.ServeHTTP)
	return nil
}

func (s *Cmd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := s.cobraCommand.Context()
	out.Raise(ctx, out.NewHTTPRequest(r))

	s.handler.ServeHTTP(w, r)
	events.Success(ctx)
}
