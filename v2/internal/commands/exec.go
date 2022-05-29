package commands

import (
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/raphaelreyna/oneshot/v2/internal/cgi"
	"github.com/raphaelreyna/oneshot/v2/internal/server"
	"github.com/raphaelreyna/oneshot/v2/internal/summary"
	"github.com/spf13/cobra"
)

func init() {
	x := execCmd{
		header: make(http.Header),
	}
	root.AddCommand(x.command())
}

type execCmd struct {
	cobraCommand *cobra.Command
	handler      *cgi.Handler
	header       http.Header
}

func (c *execCmd) command() *cobra.Command {
	c.cobraCommand = &cobra.Command{
		Use:  "exec command",
		RunE: c.runE,
	}

	flags := c.cobraCommand.Flags()
	flags.Bool("enforce-cgi", false, "The exec must conform to the CGI.")
	flags.StringSliceP("env", "e", []string{}, "Set an environment variable")
	flags.String("dir", "", "Set the working directory")
	flags.String("stderr", "", "Where to send exec stderr")
	flags.Bool("replace-headers", false, "Allow exec to replace header values")

	return c.cobraCommand
}

func (c *execCmd) runE(cmd *cobra.Command, args []string) error {
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
		Header:        headerFromStringSlice(headerSlice),
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
		markForClose(ctx, handlerConf.Stderr.(io.WriteCloser))
	} else {
		handlerConf.Stderr = cmd.ErrOrStderr()
	}

	handler, err := cgi.NewHandler(handlerConf)
	if err != nil {
		return err
	}

	c.handler = handler
	srvr := server.NewServer(c)
	setServer(ctx, srvr)
	return nil
}

func (s *execCmd) ServeHTTP(w http.ResponseWriter, r *http.Request) (*summary.Request, error) {
	var (
		cmd          = s.cobraCommand
		flags        = cmd.Flags()
		allowBots, _ = flags.GetBool("allow-bots")

		sr = summary.NewRequest(r)
	)

	// Filter out requests from bots, iMessage, etc. by checking the User-Agent header for known bot headers
	if headers, exists := r.Header["User-Agent"]; exists && !allowBots {
		if isBot(headers) {
			w.WriteHeader(http.StatusOK)
			return sr, errors.New("bot")
		}
	}

	s.handler.ServeHTTP(w, r)
	return sr, nil
}

func (s *execCmd) ServeExpiredHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("expired hello from server"))
}
