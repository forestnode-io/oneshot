package rproxy

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

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
	host         string
	config       *Configuration
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:     "reverse-proxy host",
		Aliases: []string{"rproxy"},
		Short:   "Reverse proxy all requests to the specified host",
		Long:    `Reverse proxy all requests to the specified host. The host may be a URL or a host:port combination.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			c.config.MergeFlags()
			return c.config.Validate()
		},
		RunE: c.setHandlerFunc,
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

	c.config.SetFlags(c.cobraCommand, c.cobraCommand.Flags())

	return c.cobraCommand
}

func (c *Cmd) setHandlerFunc(cmd *cobra.Command, args []string) error {
	var (
		ctx = cmd.Context()

		host      = args[0]
		spoofHost = c.config.SpoofHost
	)

	output.IncludeBody(ctx)

	hostURL, err := url.Parse(host)
	if err != nil {
		return err
	}

	var spoofedHostURL *url.URL
	if c.config.MatchHost {
		spoofedHostURL = hostURL
	}
	if spoofHost != "" {
		spoofHost = strings.TrimPrefix(spoofHost, "http://")
		spoofHost = strings.TrimPrefix(spoofHost, "https://")
		if idx := strings.Index(spoofHost, "/"); -1 < idx {
			spoofHost = spoofHost[:idx]
		}
		spoofedHostURL = &url.URL{
			Host: spoofHost,
		}
	}

	c.host = host

	if spoofedHostURL != nil {
		if len(c.config.RequestHeader) == 0 {
			c.config.RequestHeader = make(map[string][]string)
		}
		c.config.RequestHeader["Host"] = []string{spoofedHostURL.Host}
	}

	rp := httputil.NewSingleHostReverseProxy(hostURL)
	rp.ModifyResponse = func(resp *http.Response) error {
		ctx := c.cobraCommand.Context()
		originalHeader := resp.Header.Clone()

		events.Raise(ctx, &events.HTTPResponse{
			StatusCode: resp.StatusCode,
			Header:     originalHeader,
		})

		if 0 < len(c.config.ResponseHeader) {
			for k, v := range c.config.ResponseHeader {
				resp.Header[k] = v
			}
		}
		if c.config.StatusCode != 0 {
			resp.StatusCode = c.config.StatusCode
		}
		return nil
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		ctx := c.cobraCommand.Context()
		events.Raise(ctx, output.NewHTTPRequest(r))

		if 0 < len(c.config.RequestHeader) {
			for k, v := range c.config.RequestHeader {
				r.Header[k] = v
			}
		}

		if spoofedHostURL != nil {
			r.Host = spoofedHostURL.Host
			r.URL = spoofedHostURL
		}

		if c.config.Method != "" {
			r.Method = strings.ToUpper(c.config.Method)
		}

		var jsonOutput bool
		if format, _ := output.GetFormatAndOpts(ctx); format == "json" {
			jsonOutput = true
		}

		if c.config.Tee || jsonOutput {
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
