package rproxy

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	"github.com/raphaelreyna/oneshot/v2/pkg/events"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

func New() *Cmd {
	return &Cmd{
		requestHeaders:  make(http.Header),
		responseHeaders: make(http.Header),
	}
}

type Cmd struct {
	cobraCommand *cobra.Command

	requestHeaders  http.Header
	responseHeaders http.Header
	statusCode      int
	host            string
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.requestHeaders == nil {
		c.requestHeaders = make(http.Header)
	}
	if c.responseHeaders == nil {
		c.responseHeaders = make(http.Header)
	}
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:     "reverse-proxy host",
		Aliases: []string{"rproxy"},
		Short:   "Reverse proxy all requests to the specified host",
		Long:    `Reverse proxy all requests to the specified host. The host may be a URL or a host:port combination.`,
		RunE:    c.setHandlerFunc,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("proxy host required")
			}
			if 1 < len(args) {
				return errors.New("too many arguments, only 1 host may be used")
			}
			return nil
		},
	}

	flags := c.cobraCommand.Flags()
	flags.Int("status-code", http.StatusOK, "HTTP status code to send with the proxied response.")

	flags.String("method", "", "HTTP method to send with the proxied request.")

	flags.Bool("match-host", false, `The 'Host' header will be set to match the host being reverse-proxied to.`)

	flags.Bool("tee", false, `Send a copy of the proxied response to the console.`)

	flags.String("spoof-host", "", `Spoof the request host, the 'Host' header will be set to this value.
This Flag is ignored if the --match-host flag is set.`)

	flags.StringSlice("request-header", nil, `Header to send with the proxied request. Can be specified multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)

	flags.StringSlice("response-header", nil, `Header to send to send with the proxied response. Can be specified multiple times.
Format: <HEADER NAME>=<HEADER VALUE>`)

	return c.cobraCommand
}

func (c *Cmd) setHandlerFunc(cmd *cobra.Command, args []string) error {
	var (
		ctx = cmd.Context()

		flags              = c.cobraCommand.Flags()
		statusCode, _      = flags.GetInt("status-code")
		requestHeaders, _  = flags.GetStringSlice("request-header")
		responseHeaders, _ = flags.GetStringSlice("response-header")
		matchHost, _       = flags.GetBool("match-host")
		spoofHost, _       = flags.GetString("spoof-host")
		method, _          = flags.GetString("method")
		tee, _             = flags.GetBool("tee")

		host = args[0]
	)

	output.IncludeBody(ctx)

	hostURL, err := url.Parse(host)
	if err != nil {
		return err
	}

	var spoofedHostURL *url.URL
	if matchHost {
		spoofedHostURL = hostURL
	}
	if spoofHost == "" {
		spoofedHostURL, err = url.Parse(spoofHost)
		if err != nil {
			return err
		}
	}

	c.statusCode = statusCode
	c.host = host
	c.requestHeaders = oneshothttp.HeaderFromStringSlice(requestHeaders)
	c.responseHeaders = oneshothttp.HeaderFromStringSlice(responseHeaders)

	rp := httputil.NewSingleHostReverseProxy(hostURL)
	rp.ModifyResponse = func(resp *http.Response) error {
		ctx := c.cobraCommand.Context()
		originalHeader := http.Header{}
		for k, v := range resp.Header {
			originalHeader[k] = v
		}
		events.Raise(ctx, &events.HTTPResponse{
			StatusCode: resp.StatusCode,
			Header:     originalHeader,
		})

		if c.responseHeaders != nil {
			for k, v := range c.responseHeaders {
				resp.Header[k] = v
			}
		}
		if c.statusCode != 0 {
			resp.StatusCode = c.statusCode
		}
		return nil
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		ctx := c.cobraCommand.Context()
		events.Raise(ctx, output.NewHTTPRequest(r))

		if c.requestHeaders != nil {
			for k, v := range c.requestHeaders {
				r.Header[k] = v
			}
		}

		if spoofedHostURL != nil {
			r.Host = spoofedHostURL.Host
			r.URL = spoofedHostURL
		}

		if method != "" {
			r.Method = strings.ToUpper(method)
		}

		var jsonOutput bool
		if format, _ := output.GetFormatAndOpts(ctx); format == "json" {
			jsonOutput = true
		}

		if tee || jsonOutput {
			bw, getBufByte := output.NewBufferedWriter(ctx, w)

			ww := bw.(http.ResponseWriter)
			rp.ServeHTTP(ww, r)

			events.Raise(ctx, &events.File{
				Content: getBufByte,
			})
		} else {
			rp.ServeHTTP(w, r)
		}

		events.Success(ctx)
	}

	commands.SetHTTPHandlerFunc(ctx, handler)
	return nil
}
