package cmd

func SetFlags() {
	RootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Version for oneshot.")

	RootCmd.Flags().StringVarP(&port, "port", "p", "8080", `Port to bind to.
`,
	)
	RootCmd.Flags().DurationVarP(&timeout, "timeout", "t", 0, `How long to wait for client.
A value of zero will cause oneshot to wait indefinitely.`,
	)
	RootCmd.Flags().BoolVarP(&noInfo, "quiet", "q", false, `Don't show info messages.
Use -Q, --silent instead to suppress error messages as well.`,
	)
	RootCmd.Flags().BoolVarP(&noInfo, "silent", "Q", false, `Don't show info and error messages.
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
If a password is not also provided using either the -P, --password;
-W, --hidden-password; or -w, --password-file flags then the client may enter any password.`,
	)
	RootCmd.Flags().StringVarP(&password, "password", "P", "", `Password for basic authentication.
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
}
