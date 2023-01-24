package root

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/pflag"
	"golang.org/x/term"
)

func tlsCertAndKey(flags *pflag.FlagSet) (string, string, error) {
	var (
		tlsCert, _ = flags.GetString("tls-cert")
		tlsKey, _  = flags.GetString("tls-key")
	)
	if tlsCert != "" && tlsKey == "" {
		return "", "", fmt.Errorf("tls cert provided without a key")
	}

	if tlsKey != "" && tlsCert == "" {
		return "", "", fmt.Errorf("tls key provided without a cert")
	}

	return tlsCert, tlsKey, nil
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
	pflags := r.PersistentFlags()
	pflags.BoolP("quiet", "q", false, "Silence all messages.")

	pflags.String("tls-cert", "", `Certificate file to use for HTTPS.
		If the empty string ("") is passed to both this flag and --tls-key, then oneshot will generate, self-sign and use a TLS certificate/key pair.
		Key file must also be provided using the --tls-key flag.
		See also: --tls-key`)

	pflags.String("tls-key", "", `Key file to use for HTTPS.
		If the empty string ("") is passed to both this flag and --tls-cert, then oneshot will generate, self-sign and use a TLS certificate/key pair.
		Cert file must also be provided using the --tls-cert flag.
		See also: --tls-cert`)

	pflags.String("host", "", `Host specifies the TCP address for the server to listen on.
		See also: -p, --port`)
	pflags.StringP("port", "p", "8080", `Port to bind to.
		See also: --host`)

	pflags.StringP("username", "u", "", `Username for basic authentication.
		If an empty username ("") is set then a random, easy to remember username will be used.
		If a password is not also provided using either the -P, --password flag ; -W, --prompt-password; or --password-file flags then the client may enter any password.
		See also: -P, --password ; -W --prompt-password ; --password-file`)

	pflags.StringP("password", "P", "", `Password for basic authentication.
		If an empty password ("") is set then a random secure will be used.
		If a username is not also provided using the -U, --username flag then the client may enter any username.
		If either the -W, --prompt-password or --password-file flags are set, this flag will be ignored.
		See also: -U, --username ; -W --prompt-password ; --password-file`)

	pflags.BoolP("prompt-password", "W", false, `Prompt for password for basic authentication.
		If a username is not also provided using the -U, --username flag then the client may enter any username.
		See also: -U, --username ; -P --password ; --password-file`)

	pflags.String("password-file", "", `File containing password for basic authentication.
		If a username is not also provided using the -U, --username flag then the client may enter any username.
		If the -W, --prompt-password flag is set, this flags will be ignored.
		See also: -U, --username ; -P --password ; -W --prompt-password`)

	pflags.Duration("timeout", 0, `How long to wait for client. A value of zero will cause oneshot to wait indefinitely.`)

	pflags.VarP(&r.outFlag, "output", "o", `Set output format. Valid formats are: json[=opts].
		Valid json opts are:
			- pretty
				Enables tabbed, pretty printed json.
			- include-file-contents
				Includes the contents of files in the json output.
				This is on by default when sending from stdin or receiving to stdout.
			- exclude-file-contents
				Excludes the contents of files in the json output.
				This is on by default when sending or receiving to or from disk.
`)
	pflags.Bool("allow-bots", false, `Don't block bots.`)
	pflags.StringArrayP("header", "H", nil, "HTTP header to send to client.\nSetting a value for 'Content-Type' will override the -M, --mime flag.")

	pflags.BoolP("qr-code", "Q", false, `Generate QR codes for connection URLs`)
	pflags.Bool("no-color", false, `Don't use color`)
}

type outputFormatFlagArg struct {
	format string
	opts   []string
}

func (o *outputFormatFlagArg) String() string {
	s := o.format
	if 0 < len(o.opts) {
		s += "=" + strings.Join(o.opts, ",")
	}
	return s
}

func (o *outputFormatFlagArg) Set(v string) error {
	switch {
	case strings.HasPrefix(v, "json"):
		o.format = "json"
		parts := strings.Split(v, "=")
		if len(parts) < 2 {
			return nil
		}
		o.opts = strings.Split(parts[1], ",")
		return nil
	}
	return errors.New(`must be "json[=opts...]"`)
}

func (o *outputFormatFlagArg) Type() string {
	return "string"
}
