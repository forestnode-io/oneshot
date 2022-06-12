package commands

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/raphaelreyna/oneshot/v2/internal/network"
	"github.com/raphaelreyna/oneshot/v2/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

var root rootCommand

func init() {
	root.garbageFiles = make([]string, 0)
	root.Use = "oneshot"
	root.PersistentPreRunE = root.persistentPreRunE
	root.PersistentPostRunE = root.persistentPostRunE

	root.setFlags()
}

func ExecuteContext(ctx context.Context) error {
	ctx = withServer(ctx, &root.server)
	ctx = withClosers(ctx, &root.closers)
	ctx = withFileGarbageCollection(ctx, &root.garbageFiles)

	defer func() {
		for _, closer := range root.closers {
			closer.Close()
		}
		for _, path := range root.garbageFiles {
			os.Remove(path)
		}
	}()

	return root.ExecuteContext(ctx)
}

type rootCommand struct {
	cobra.Command
	server       *server.Server
	garbageFiles []string
	closers      []io.Closer
	middleware   server.Middleware
}

func (r *rootCommand) persistentPreRunE(cmd *cobra.Command, args []string) error {
	var (
		flags                  = cmd.Flags()
		allowBots, _           = flags.GetBool("allow-bots")
		unauthenticatedHandler = http.HandlerFunc(nil)
		uname, passwd, err     = usernamePassword(flags)
	)
	if err != nil {
		return err
	}

	r.middleware = r.middleware.
		Chain(botsMiddleware(allowBots)).
		Chain(authMiddleware(unauthenticatedHandler, uname, passwd))

	return nil
}

func (r *rootCommand) persistentPostRunE(cmd *cobra.Command, args []string) error {
	var (
		ctx = cmd.Context()

		flags              = cmd.Flags()
		host, _            = flags.GetString("host")
		portNum, _         = flags.GetString("port")
		jopts, jsonFlagErr = flags.GetString("json")
		wantsJSON          = jsonFlagErr == nil && jopts != ""
		timeout, _         = flags.GetDuration("timeout")
	)

	defer func() {
		for _, fp := range r.garbageFiles {
			_ = os.Remove(fp)
		}
	}()

	if portNum != "" {
		portNum = ":" + portNum
	}

	if strings.Contains(jopts, "include-body") {
		r.server.BufferRequests()
	}

	r.server.SetTimeoutDuration(timeout)
	r.server.AddMiddleware(r.middleware)

	host += portNum
	l, err := net.Listen("tcp", host)
	if err != nil {
		return err
	}
	defer l.Close()

	stdout := cmd.OutOrStdout()
	if !wantsJSON {
		if host == "" {
			addrs, err := network.HostAddresses()
			if err != nil {
				return err
			}

			fmt.Fprintln(stdout, "listening on: ")
			for _, addr := range addrs {
				fmt.Fprintf(stdout, "\t- http://%s%s\n", addr, portNum)
			}
		} else {
			fmt.Fprintf(stdout, "listening on: http://localhost%s\n", portNum)
		}
	}

	if err := r.server.Serve(ctx, l); err != nil {
		return err
	}

	summary := r.server.Summary()
	if wantsJSON {
		summary.WriteJSON(stdout, strings.Contains(jopts, "pretty"))
	} else {
		summary.WriteHuman(stdout)
	}

	return nil
}

func usernamePassword(flags *pflag.FlagSet) (string, string, error) {
	var (
		username, _ = flags.GetString("username")
		password, _ = flags.GetString("password")
	)

	if username != "" && password != "" {
		return username, password, nil
	}

	if path, _ := flags.GetString("password-file"); path != "" {
		passwdBytes, err := os.ReadFile(path)
		if err != nil {
			return "", "", err
		}
		password = string(passwdBytes)
		return username, password, nil
	}

	if x, _ := flags.GetBool("prompt-password"); x {
		fmt.Print("Enter Password: ")
		passwdBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", "", err
		}
		password = strings.TrimSpace(string(passwdBytes))
	}

	return username, password, nil
}

func (r *rootCommand) setFlags() {
	pflags := root.PersistentFlags()
	/*

			pflags.String("host", "", "Host specifies the TCP address for the server to listen on.")


			pflags.BoolP("quiet", "q", false, "Don't show info messages.")

			pflags.BoolP("silent", "Q", false, "Supress all messages, including errors.")

			pflags.BoolP("ss-tls", "T", false, `Generate and use a self-signed TLS certificate/key pair for HTTPS.
		A new certificate/key pair is generated for each running instance of oneshot.
		To use your own certificate/key pair, use the --tls-cert and --tls-key flags.`)

			pflags.String("tls-cert", "", `Certificate file to use for HTTPS.
		If the empty string ("") is passed to both this flag and --tls-key, then oneshot will generate, self-sign and use a TLS certificate/key pair.
		Key file must also be provided using the --tls-key flag.
		See also: --tls-key ; -T, --ss-tls`)

			pflags.String("tls-key", "", `Key file to use for HTTPS.
		If the empty string ("") is passed to both this flag and --tls-cert, then oneshot will generate, self-sign and use a TLS certificate/key pair.
		Cert file must also be provided using the --tls-cert flag.
		See also: --tls-cert ; -T, --ss-tls`)

			pflags.BoolP("mdns", "M", false, "Register oneshot as an mDNS (bonjour/avahi) service.")

	*/
	pflags.StringP("port", "p", "8080", "Port to bind to.")

	pflags.String("username", "", `Username for basic authentication.
		If an empty username ("") is set then a random, easy to remember username will be used.
		If a password is not also provided using either the -P, --password flag ; -W, --hidden-password; or -w, --password-file flags then the client may enter any password.`)

	pflags.String("password", "", `Password for basic authentication.
		If an empty password ("") is set then a random secure will be used.
		If a username is not also provided using the -U, --username flag then the client may enter any username.
		If either the -W, --hidden-password or -w, --password-file flags are set, this flag will be ignored.`)

	pflags.Bool("prompt-password", false, "Prompt for password for basic authentication.\nIf a username is not also provided using the -U, --username flag then the client may enter any username.")

	pflags.String("password-file", "", `File containing password for basic authentication.
		If a username is not also provided using the -U, --username flag then the client may enter any username.
		If the -W, --hidden-password flag is set, this flags will be ignored.`)

	pflags.Duration("timeout", 0, `How long to wait for client. A value of zero will cause oneshot to wait indefinitely.`)

	pflags.String("json", "", `Enable JSON output. Options: include-body`)
	pflags.Lookup("json").NoOptDefVal = "true"
	pflags.Bool("allow-bots", false, `Don't block bots.`)
	pflags.StringArrayP("header", "H", nil, "HTTP header to send to client.\nSetting a value for 'Content-Type' will override the -M, --mime flag.")
}

type serverKey struct{}

type fileGCKey struct{}

func withServer(ctx context.Context, sdp **server.Server) context.Context {
	return context.WithValue(ctx, serverKey{}, sdp)
}

func withFileGarbageCollection(ctx context.Context, files *[]string) context.Context {
	return context.WithValue(ctx, fileGCKey{}, files)
}

func setServer(ctx context.Context, s *server.Server) {
	if sdp, ok := ctx.Value(serverKey{}).(**server.Server); ok {
		*sdp = s
	}
}

func markFilesAsGarbage(ctx context.Context, filePaths ...string) {
	if files, ok := ctx.Value(fileGCKey{}).(*[]string); ok {
		*files = append(*files, filePaths...)
	}
}

type closerKey struct{}

func withClosers(ctx context.Context, closers *[]io.Closer) context.Context {
	return context.WithValue(ctx, closerKey{}, closers)
}

func markForClose(ctx context.Context, closer io.Closer) {
	if closers, ok := ctx.Value(closerKey{}).(*[]io.Closer); ok {
		*closers = append(*closers, closer)
	}
}
