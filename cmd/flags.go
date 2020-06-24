package cmd

func SetFlags() {
	RootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, `Version for oneshot.`)

	RootCmd.Flags().BoolVarP(&exitOnFail, "exit-on-fail", "F", false, `Exit as soon as client disconnects regardless if file was transferred succesfully.
By default, oneshot will exit once the client has downloaded the entire file.
If using authentication, setting this flag will cause oneshot to exit if client provides wrong / no credentials.
If set, once the first client connects, all others will receive a 410 Gone status immediately;
otherwise, client waits in a queue and is served if all previous clients fail or drop out.`,
	)

	RootCmd.Flags().StringVarP(&port, "port", "p", "8080", `Port to bind to.`)

	RootCmd.Flags().DurationVarP(&timeout, "timeout", "t", 0, `How long to wait for client.
A value of zero will cause oneshot to wait indefinitely.`,
	)

	RootCmd.Flags().BoolVarP(&noInfo, "quiet", "q", false, `Don't show info messages.
Use -Q, --silent instead to suppress error messages as well.`,
	)

	RootCmd.Flags().BoolVarP(&noError, "silent", "Q", false, `Don't show info and error messages.
Use -q, --quiet instead to suppress info messages only.`,
	)

	RootCmd.Flags().BoolVarP(&noDownload, "no-download", "D", false, `Don't trigger browser download client side.
If set, the "Content-Disposition" header used to trigger downloads in the clients browser won't be sent.`,
	)

	RootCmd.Flags().StringVarP(&fileName, "name", "n", "", `Name of file presented to client.
If not set, either a random name or the name of the file will be used,
depending on if a file was given.`,
	)

	RootCmd.Flags().StringVarP(&fileExt, "ext", "e", "", `Extension of file presented to client.
If not set, either no extension or the extension of the file will be used,
depending on if a file was given.`,
	)

	RootCmd.Flags().StringVarP(&fileMime, "mime", "m", "", `MIME type of file presented to client.
If not set, either no MIME type or the mime/type of the file will be user,
depending on of a file was given.`,
	)

	RootCmd.Flags().StringVar(&certFile, "tls-cert", "", `Certificate file to use for HTTPS.
Key file must also be provided using the --tls-key flag.`,
	)

	RootCmd.Flags().StringVar(&keyFile, "tls-key", "", `Key file to use for HTTPS.
Cert file must also be provided using the --tls-cert flag.`,
	)

	RootCmd.Flags().StringVarP(&username, "username", "U", "", `Username for basic authentication.
If an empty username ("") is set then a random, easy to remember username will be used.
If a password is not also provided using either the -P, --password flag , then the client may enter any password;
-W, --hidden-password; or -w, --password-file flags then the client may enter any password.`,
	)

	RootCmd.Flags().StringVarP(&password, "password", "P", "", `Password for basic authentication.
If an empty password ("") is set then a random secure will be used.
If a username is not also provided using the -U, --username flag then the client may enter any username.
If either the -W, --hidden-password or -w, --password-file flags are set, this flag will be ignored.`,
	)

	RootCmd.Flags().StringVarP(&passwordFile, "password-file", "w", "", `File containing password for basic authentication.
If a username is not also provided using the -U, --username flag then the client may enter any username.
If the -W, --hidden-password flag is set, this flags will be ignored.`,
	)
	RootCmd.Flags().BoolVarP(&passwordHidden, "hidden-password", "W", false, `Prompt for password for basic authentication.
If a username is not also provided using the -U, --username flag then the client may enter any username.
Takes precedence over the -w, --password-file flag`,
	)

	RootCmd.Flags().BoolVarP(&cgi, "cgi", "c", false, `Run the given file in a forgiving CGI environment.
Setting this flag will override the -u, --upload flag.
See also: -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr`,
	)

	RootCmd.Flags().BoolVarP(&cgiStrict, "cgi-strict", "C", false, `Run the given file in a CGI environment.
Setting this flag overrides the -c, --cgi flag and acts as a modifier to the -S, --shell-command flag.
If this flag is set, the file passed to oneshot will be run in a strict CGI environment; i.e. if the executable attempts to send invalid headers, oneshot will exit with an error.
If you instead wish to simply send an executables stdout without worrying about setting headers, use the -c, --cgi flag.
If the -S, --shell-command flag is used to pass a command, this flag has no effect.
Setting this flag will override the -u, --upload flag.
See also: -c, --cgi ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr`,
	)

	RootCmd.Flags().BoolVarP(&shellCommand, "shell-command", "S", false, `Run a shell command in a flexible CGI environment.
If you wish to run the command in a strict CGI environment where oneshot exits upon detecting invalid headers, use the -C, --strict-cgi flag as well.
If this flag is used to pass a shell command, then any file passed to oneshot will be ignored.
Setting this flag will override the -u, --upload flag.
See also: -c, --cgi ; -C, --cgi-strict ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr`,
	)

	RootCmd.Flags().StringVarP(&shell, "shell", "s", "/bin/sh", `Shell that should be used when running a shell command.
Setting this flag does nothing if the -S, --shell-command flag is not set.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr`,
	)

	RootCmd.Flags().BoolVarP(&replaceHeaders, "replace-header", "R", false, `HTTP header to send to client.
To allow executable to override header see the --replace flag.
Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
Must be in the form 'KEY: VALUE'.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -H, --header ; -E, --env ; --cgi-stderr`,
	)

	RootCmd.Flags().StringArrayVarP(&rawHeaders, "header", "H", nil, `HTTP header to send to client.
Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
To allow executable to override header see the -R, --replace-headers flag.
Must be in the form 'KEY: VALUE'.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -E, --env ; --cgi-stderr`,
	)

	RootCmd.Flags().StringArrayVarP(&envVars, "env", "E", nil, `Environment variable to pass on to the executable.
Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
Must be in the form 'KEY=VALUE'.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr`,
	)

	RootCmd.Flags().StringVar(&cgiStderr, "cgi-stderr", "", `Where to redirect executable's stderr when running in CGI mode.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr`,
	)

	RootCmd.Flags().StringVarP(&dir, "dir", "d", "", `Working directory for the executable or when saving files.
Defaults to where oneshot was called.
Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr`,
	)

	RootCmd.Flags().BoolVarP(&upload, "upload", "u", false, `Receive a file from the client.
Setting this flag will cause oneshot to serve up a minimalistic web-page that prompts the client to upload a file.
By default if no path argument is given, the file will be sent to standard out (nothing else will be printed to standard out, this is useful for when you wish to pipe or redirect the file uploaded by the client).
If a path to a directory is given as an argument (or the -d, --dir flag is set), oneshot will save the file to that directory using either the files original name or the one set by the -n, --name flag.
If both the -d, --dir flag is set and a path is given as an argument, then the path from -d, --dir is prepended to the one from the argument.

Example: Running "oneshot -u -d /foo ./bar/baz" will result in the clients uploaded file being saved to directory /foo/bar/baz.

This flag actually exposes an upload API as well.
Oneshot will save either the entire body, or first file part (if the Content-Type is set to multipart/form-data) of any POST request sent to "/"

Example: Running "curl -d 'Hello World!' localhost:8080" will send 'Hello World!' to oneshot.
`,
	)

	RootCmd.Flags().StringVarP(&archiveMethod, "archive-method", "a", "tar.gz", `Which archive method to use when sending directories.
Recognized values are "zip" and "tar.gz", any unrecognized values will default to "tar.gz".`,
	)
}
