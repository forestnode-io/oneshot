package cmd

import (
	"fmt"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"github.com/spf13/cobra"
	"log"
	"net"
	"os"
	"strings"
)

type App struct {
	hostname string
	ips      []string
	cmd      *cobra.Command
	infoLog  *log.Logger
	errLog   *log.Logger
	conf     *conf

	server *server.Server
}

// Create an unconfigured app instance
func newApp() (*App, error) {
	var err error
	app := &App{}

	// Get hostname
	app.hostname, err = os.Hostname()
	if err != nil {
		return nil, err
	}

	// Get ip addresses
	app.ips = []string{}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	// Filter out loopback and ipv6 addresses
	var home string
	for _, addr := range addrs {
		saddr := addr.String()

		if strings.Contains(saddr, "::") {
			continue
		}

		parts := strings.Split(saddr, "/")
		ip := parts[0]

		// Remove localhost since whats the point in sharing with yourself? (usually)
		if parts[0] == "127.0.0.1" || parts[0] == "localhost" {
			home = ip
			continue
		}

		app.ips = append(app.ips, ip)
	}

	// If no addresses other than loopbacks are available, use those
	if len(app.ips) == 0 {
		app.ips = append(app.ips, home)
	}

	// Create the cobra command
	app.cmd = &cobra.Command{
		Use:     "oneshot [flags]... [file|dir]",
		Version: fmt.Sprintf(": %s\ndate : %s\nauthor : Raphael Reyna", version, date),
		Short:   "A single-fire first-come-first-serve HTTP server.",
		Long: `Transfer files and data easily between your computer and any browser or HTTP client.
The first client to connect is given the file or uploads a file, all others receive an HTTP 410 Gone response code.
Directories will automatically be archived before being sent (see -a, --archive-method for more information).
`,
		Run: run,
	}

	return app, nil
}

func (a *App) setFlags() {
	a.conf = newConf()
	flags := a.cmd.Flags()

	flags.BoolVarP(&versionFlag, "version", "v", false, `Version and other info.`)

	flags.BoolVarP(a.conf.exitOnFail, "exit-on-fail", "F", false, `Exit as soon as client disconnects regardless if file was transferred succesfully.
By default, oneshot will exit once the client has downloaded the entire file.
If using authentication, setting this flag will cause oneshot to exit if client provides wrong / no credentials.
If set, once the first client connects, all others will receive a 410 Gone status immediately;
otherwise, client waits in a queue and is served if all previous clients fail or drop out.`,
	)

	flags.StringVarP(a.conf.port, "port", "p", "8080", `Port to bind to.`)

	flags.DurationVarP(a.conf.timeout, "timeout", "t", 0, `How long to wait for client.
A value of zero will cause oneshot to wait indefinitely.`,
	)

	flags.BoolVarP(a.conf.noInfo, "quiet", "q", false, `Don't show info messages.
Use -Q, --silent instead to suppress error messages as well.`,
	)

	flags.BoolVarP(a.conf.noError, "silent", "Q", false, `Don't show info and error messages.
Use -q, --quiet instead to suppress info messages only.`,
	)

	flags.BoolVarP(a.conf.noDownload, "no-download", "D", false, `Don't trigger browser download client side.
If set, the "Content-Disposition" header used to trigger downloads in the clients browser won't be sent.`,
	)

	flags.StringVarP(a.conf.fileName, "name", "n", "", `Name of file presented to client.
If not set, either a random name or the name of the file will be used,
depending on if a file was given.`,
	)

	flags.StringVarP(a.conf.fileExt, "ext", "e", "", `Extension of file presented to client.
If not set, either no extension or the extension of the file will be used,
depending on if a file was given.`,
	)

	flags.StringVarP(a.conf.fileMime, "mime", "m", "", `MIME type of file presented to client.
If not set, either no MIME type or the mime/type of the file will be user,
depending on of a file was given.`,
	)

	flags.StringVar(a.conf.certFile, "tls-cert", "", `Certificate file to use for HTTPS.
If the empty string ("") is passed to both this flag and --tls-key, then oneshot will generate, self-sign and use a TLS certificate/key pair.
Key file must also be provided using the --tls-key flag.
See also: --tls-key ; -T, --ss-tls`,
	)

	flags.StringVar(a.conf.keyFile, "tls-key", "", `Key file to use for HTTPS.
If the empty string ("") is passed to both this flag and --tls-cert, then oneshot will generate, self-sign and use a TLS certificate/key pair.
Cert file must also be provided using the --tls-cert flag.
See also: --tls-cert ; -T, --ss-tls`,
	)

	flags.StringVarP(a.conf.username, "username", "U", "", `Username for basic authentication.
If an empty username ("") is set then a random, easy to remember username will be used.
If a password is not also provided using either the -P, --password flag ; -W, --hidden-password; or -w, --password-file flags then the client may enter any password.`,
	)

	flags.StringVarP(a.conf.password, "password", "P", "", `Password for basic authentication.
If an empty password ("") is set then a random secure will be used.
If a username is not also provided using the -U, --username flag then the client may enter any username.
If either the -W, --hidden-password or -w, --password-file flags are set, this flag will be ignored.`,
	)

	flags.StringVarP(a.conf.passwordFile, "password-file", "w", "", `File containing password for basic authentication.
If a username is not also provided using the -U, --username flag then the client may enter any username.
If the -W, --hidden-password flag is set, this flags will be ignored.`,
	)
	flags.BoolVarP(a.conf.passwordHidden, "hidden-password", "W", false, `Prompt for password for basic authentication.
If a username is not also provided using the -U, --username flag then the client may enter any username.
Takes precedence over the -w, --password-file flag`,
	)

	flags.BoolVarP(a.conf.cgi, "cgi", "c", false, `Run the given file in a forgiving CGI environment.
Setting this flag will override the -u, --upload flag.
See also: -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr`,
	)

	flags.BoolVarP(a.conf.cgiStrict, "cgi-strict", "C", false, `Run the given file in a CGI environment.
Setting this flag overrides the -c, --cgi flag and acts as a modifier to the -S, --shell-command flag.
If this flag is set, the file passed to oneshot will be run in a strict CGI environment; i.e. if the executable attempts to send invalid headers, oneshot will exit with an error.
If you instead wish to simply send an executables stdout without worrying about setting headers, use the -c, --cgi flag.
If the -S, --shell-command flag is used to pass a command, this flag has no effect.
Setting this flag will override the -u, --upload flag.
See also: -c, --cgi ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr`,
	)

	flags.BoolVarP(a.conf.shellCommand, "shell-command", "S", false, `Run a shell command in a flexible CGI environment.
If you wish to run the command in a strict CGI environment where oneshot exits upon detecting invalid headers, use the -C, --strict-cgi flag as well.
If this flag is used to pass a shell command, then any file passed to oneshot will be ignored.
Setting this flag will override the -u, --upload flag.
See also: -c, --cgi ; -C, --cgi-strict ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr`,
	)

	flags.StringVarP(a.conf.shell, "shell", "s", shellDefault, `Shell that should be used when running a shell command.
Setting this flag does nothing if the -S, --shell-command flag is not set.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr`,
	)

	flags.BoolVarP(a.conf.replaceHeaders, "replace-header", "R", false, `HTTP header to send to client.
To allow executable to override header see the --replace flag.
Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
Must be in the form 'KEY: VALUE'.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -H, --header ; -E, --env ; --cgi-stderr`,
	)

	flags.StringArrayVarP(&a.conf.rawHeaders, "header", "H", nil, `HTTP header to send to client.
Setting a value for 'Content-Type' will override the -M, --mime flag.
To allow executable to override header see the -R, --replace-headers flag.
Must be in the form 'KEY: VALUE'.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -E, --env ; --cgi-stderr`,
	)

	flags.StringArrayVarP(&a.conf.envVars, "env", "E", nil, `Environment variable to pass on to the executable.
Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
Must be in the form 'KEY=VALUE'.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr`,
	)

	flags.StringVar(a.conf.cgiStderr, "cgi-stderr", "", `Where to redirect executable's stderr when running in CGI mode.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr`,
	)

	flags.StringVarP(a.conf.dir, "dir", "d", "", `Working directory for the executable or when saving files.
Defaults to where oneshot was called.
Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr`,
	)

	flags.BoolVarP(a.conf.upload, "upload", "u", false, `Receive a file, allow client to send text or upload a file to your computer.
Setting this flag will cause oneshot to serve up a minimalistic web-page that prompts the client to either upload a file or enter text.
To only allow for a file or user input and not both, see the --upload-file and --upload-input flags.
By default if no path argument is given, the file will be sent to standard out (nothing else will be printed to standard out, this is useful for when you wish to pipe or redirect the file uploaded by the client).
If a path to a directory is given as an argument (or the -d, --dir flag is set), oneshot will save the file to that directory using either the files original name or the one set by the -n, --name flag.
If both the -d, --dir flag is set and a path is given as an argument, then the path from -d, --dir is prepended to the one from the argument.
See also: --upload-file; --upload-input; -L, --no-unix-eol-norm

Example: Running "oneshot -u -d /foo ./bar/baz" will result in the clients uploaded file being saved to directory /foo/bar/baz.

This flag actually exposes an upload API as well.
Oneshot will save either the entire body, or first file part (if the Content-Type is set to multipart/form-data) of any POST request sent to "/"

Example: Running "curl -d 'Hello World!' localhost:8080" will send 'Hello World!' to oneshot.
`,
	)

	flags.StringVarP(a.conf.archiveMethod, "archive-method", "a", archiveMethodDefault, `Which archive method to use when sending directories.
Recognized values are "zip" and "tar.gz", any unrecognized values will default to "tar.gz".`,
	)

	flags.BoolVarP(a.conf.sstls, "ss-tls", "T", false, `Generate and use a self-signed TLS certificate/key pair for HTTPS.
A new certificate/key pair is generated for each running instance of oneshot.
To use your own certificate/key pair, use the --tls-cert and --tls-key flags.
See also: --tls-key ; -T, --ss-tls`,
	)

	flags.BoolVarP(a.conf.mdns, "mdns", "M", false, `Register oneshot as an mDNS (bonjour/avahi) service.`)

	flags.BoolVarP(a.conf.noUnixNorm, "no-unix-eol-norm", "L", noUnixNormDefault, `Don't normalize end-of-line chars to unix style on user input.
Most browsers send DOS style (CR+LF) end-of-line characters when submitting user form input; setting this flag to true prevents oneshot from doing the replacement CR+LF -> LF.
This flag does nothing if both the -u, --upload and --upload-input flags are not set.
See also: -u, --upload; --upload-input`,
	)

	flags.BoolVar(a.conf.uploadFile, "upload-file", false, `Receive a file, allow client to upload a file to your computer.
Setting both this flag and --upload-input is equivalent to setting the -u, --upload flag.
For more information see the -u, --upload flag documentation.
See also: --upload-input; -u, --upload`,
	)
	flags.BoolVar(a.conf.uploadInput, "upload-input", false, `Receive text from a browser.
Setting both this flag and --upload-file is equivalent to setting the -u, --upload flag.
For more information see the -u, --upload flag documentation.
See also: --upload-file; -u, --upload; -L, --no-unix-eol-norm`,
	)
}
