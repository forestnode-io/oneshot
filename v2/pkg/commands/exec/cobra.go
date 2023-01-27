package exec

import (
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/cgi"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
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
	if c.header == nil {
		c.header = make(http.Header)
	}
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "exec -- command",
		Short: "Execute a command for each request, passing in the body to stdin and returning the stdout to the client",
		Long: `Execute a command for each request, passing in the body to stdin and returning the stdout to the client.
Commands may be CGI complaint but do not have to be. CGI compliance can be enforced with the --enforce-cgi flag.`,
		RunE: c.setHandlerFunc,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("missing command")
			}

			return nil
		},
	}

	flags := c.cobraCommand.LocalFlags()
	flags.Bool("enforce-cgi", false, "The exec must conform to the CGI.")

	flags.StringSliceP("env", "e", []string{}, "Set an environment variable.")

	flags.String("dir", "", "Set the working directory.")

	flags.String("stderr", "", "Where to send exec stderr.")

	flags.Bool("replace-headers", false, "Allow command to replace header values.")

	flags.StringArrayP("header", "H", nil, `HTTP header to send to client. Can be set multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)

	return c.cobraCommand
}

func (c *Cmd) setHandlerFunc(cmd *cobra.Command, args []string) error {
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

func (c *Cmd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := c.cobraCommand.Context()
	events.Raise(ctx, output.NewHTTPRequest(r))

	bw, getBufBytes := output.NewBufferedWriter(ctx, w)
	brw := output.ResponseWriter{
		W:         w,
		BufferedW: bw,
	}
	fileReport := events.File{
		TransferStartTime: time.Now(),
	}

	c.handler.ServeHTTP(&brw, r)

	fileReport.TransferEndTime = time.Now()
	fileReport.Content = getBufBytes
	events.Raise(ctx, &fileReport)

	events.Success(ctx)
}
