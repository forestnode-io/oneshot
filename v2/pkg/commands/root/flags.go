package root

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/raphaelreyna/oneshot/v2/pkg/flagargs"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
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

	flags := pflag.NewFlagSet("Output Flags", pflag.ContinueOnError)
	flags.BoolP("quiet", "q", false, "Disable all output messages.")

	flags.VarP(&r.outFlag, "output", "o", `Set output format. Valid formats are: json[=opts].
Valid json opts are:
	- compact
		Disables tabbed, pretty printed json.
	- include-file-contents
		Includes the contents of files in the json output.
		This is on by default when sending from stdin or receiving to stdout.
	- exclude-file-contents
		Excludes the contents of files in the json output.
		This is on by default when sending or receiving to or from disk.`)

	flags.BoolP("qr-code", "Q", false, "Generate QR codes for easy connection sharing.")
	flags.Bool("no-color", false, "Disable colored output.")

	pflags.AddFlagSet(flags)
	cobra.AddTemplateFunc("flags", func(cmd *cobra.Command) *pflag.FlagSet {
		return flags
	})

	sfs := pflag.NewFlagSet("Server Flags", pflag.ContinueOnError)
	sfs.Duration("timeout", 0, `How long to wait for client.
A value of zero will cause oneshot to wait indefinitely.`)

	sfs.String("tls-cert", "", `Certificate file to use for HTTPS.
Key file must also be provided using the --tls-key flag.`)

	sfs.String("tls-key", "", `Key file to use for HTTPS.
Cert file must also be provided using the --tls-cert flag.`)

	sfs.String("host", "", `Host specifies the TCP address for the server to listen on.`)

	sfs.IntP("port", "p", 8080, `Port to bind to.`)

	sfs.Bool("allow-bots", false, "Allow bot access.")

	sfa := flagargs.Size(0)
	sfs.Var(&sfa, "max-read-size", `Maximum read size for incoming request bodies. A value of zero will cause oneshot to read until EOF.
Format is a number followed by a unit of measurement.
Valid units are: b, B, 
	Kb, KB, KiB,
	Mb, MB, MiB,
	Gb, GB, GiB,
	Tb, TB, TiB
Example: 12MB. 
	`)

	sfs.Bool("exit-on-fail", false, "Exit after a failed transfer, without waiting for a succesful one.")

	pflags.AddFlagSet(sfs)
	cobra.AddTemplateFunc("serverFlags", func(cmd *cobra.Command) *pflag.FlagSet {
		return sfs
	})

	bafs := pflag.NewFlagSet("Basic Authentication", pflag.ContinueOnError)
	bafs.StringP("username", "u", "", `Username for basic authentication.
If a password is not also provided then the client may enter any password.`)

	bafs.StringP("password", "P", "", `Password for basic authentication.
If a username is not also provided using the --username flag then the client may enter any username.
If either the --prompt-password or --password-file flags are set, this flag will be ignored.`)

	bafs.BoolP("prompt-password", "W", false, `Prompt for password when using basic authentication.
If a username is not also provided then the client may enter any username.`)

	bafs.String("password-file", "", `Path to file containing password for basic authentication.
If a username is not also provided then the client may enter any username.
If the --prompt-password flag is set, this flags will be ignored.`)

	bafs.String("unauthenticated-view", "", `Path to file that will be served to unauthenticated users.
If a username or password is not provided, this flag will be ignored.`)

	bafs.Int("unauthorized-status", http.StatusUnauthorized, `Status code that will be sent to unauthenticated users.
If a username or password is not provided, this flag will be ignored.`)

	bafs.Bool("dont-trigger-login", false, `Do not display a login dialog for unauthenticated users.
If a username or password is not provided, this flag will be ignored.`)

	pflags.AddFlagSet(bafs)
	cobra.AddTemplateFunc("basicAuthFlags", func(cmd *cobra.Command) *pflag.FlagSet {
		return bafs
	})

	cfs := pflag.NewFlagSet("CORS", pflag.ContinueOnError)
	cfs.Bool("cors", false, `Enable default CORS support.`)

	cfs.StringArray("cors-allowed-origins", nil, `Comma separated list of allowed origins.
An allowed origin may be a domain name, or a wildcard (*).
A domain name may contain a wildcard (*).`)

	cfs.StringArray("cors-allowed-headers", nil, `Comma separated list of allowed headers.
An allowed header may be a header name, or a wildcard (*).
If a wildcard (*) is used, all headers will be allowed.`)

	cfs.Int("cors-max-age", 0, `How long (in seconds) the preflight results can be cached by the client.`)

	cfs.Bool("cors-allow-credentials", false, `Allow credentials like cookies, basic auth headers, and ssl certs for CORS requests.`)

	cfs.Bool("cors-allow-private-network", false, `Allow private network for CORS requests.`)
	cfs.Int("cors-success-status", http.StatusNoContent, `Status code that will be sent to successful CORS preflight requests.`)
	pflags.AddFlagSet(cfs)
	cobra.AddTemplateFunc("corsFlags", func(cmd *cobra.Command) *pflag.FlagSet {
		return cfs
	})

	dsfs := pflag.NewFlagSet("Discovery Server Flags", pflag.ContinueOnError)
	dsfs.String("discovery-server-url", "", `URL to use for discovery server.`)
	dsfs.String("discovery-server-key", "", `Key to use for discovery server.`)
	dsfs.Bool("discovery-server-insecure", false, `Disable TLS when connecting to the discovery server.`)
	pflags.AddFlagSet(dsfs)
	cobra.AddTemplateFunc("discoveryServerFlags", func(cmd *cobra.Command) *pflag.FlagSet {
		return dsfs
	})

	p2pFS := pflag.NewFlagSet("p2p Flags", pflag.ContinueOnError)
	p2pFS.Bool("p2p", false, `Enable p2p support with default values.`)
	p2pFS.Bool("p2p-only", false, `Only allow p2p connections.`)
	p2pFS.String("p2p-config-file", "", `Path to file containing p2p configuration.`)
	p2pFS.String("p2p-discovery-dir", "", `Directory to use for p2p discovery.`)
	p2pFS.String("p2p-discovery-server-request-url", "", `URL that the discovery server will try to reserve for connecting clients.`)
	p2pFS.String("p2p-discovery-server-required-url", "", `URL that the discovery server needs to reserve for connecting clients.`)
	pflags.AddFlagSet(p2pFS)
	cobra.AddTemplateFunc("p2pFlags", func(cmd *cobra.Command) *pflag.FlagSet {
		return p2pFS
	})

	ufs := pflag.NewFlagSet("UPnP IGD Flags", pflag.ContinueOnError)
	ufs.Int("external-port", 21782, `External port to use for UPnP IGD port mapping.`)
	ufs.Duration("port-mapping-duration", 30*time.Second, `Duration to use for UPnP IGD port mapping.`)
	ufs.Duration("upnp-discovery-timeout", 45*time.Second, `Timeout to use for UPnP IGD discovery.`)
	pflags.AddFlagSet(ufs)
	cobra.AddTemplateFunc("upnpFlags", func(cmd *cobra.Command) *pflag.FlagSet {
		return ufs
	})

	r.Command.MarkFlagsMutuallyExclusive("p2p-discovery-dir", "discovery-server-url")
}

// newCorsConfig returns a new corsConfig from the given flag set.
func corsOptionsFromFlagSet(fs *pflag.FlagSet) *cors.Options {
	var opts *cors.Options
	if useCors, _ := fs.GetBool("cors"); useCors {
		opts = &cors.Options{}
	}

	fs.Visit(func(f *pflag.Flag) {
		switch f.Name {
		case "cors-allowed-origins":
			if opts == nil {
				opts = &cors.Options{}
			}
			opts.AllowedOrigins = f.Value.(pflag.SliceValue).GetSlice()
		case "cors-allowed-headers":
			if opts == nil {
				opts = &cors.Options{}
			}
			opts.AllowedHeaders = f.Value.(pflag.SliceValue).GetSlice()
		case "cors-max-age":
			if opts == nil {
				opts = &cors.Options{}
			}
			opts.MaxAge, _ = fs.GetInt("cors-max-age")
		case "cors-allow-credentials":
			if opts == nil {
				opts = &cors.Options{}
			}
			opts.AllowCredentials, _ = fs.GetBool("cors-allow-credentials")
		case "cors-allow-private-network":
			if opts == nil {
				opts = &cors.Options{}
			}
			opts.AllowPrivateNetwork, _ = fs.GetBool("cors-allow-private-network")
		case "cors-success-status":
			if opts == nil {
				opts = &cors.Options{}
			}
			opts.OptionsSuccessStatus, _ = fs.GetInt("cors-success-status")
		}
	})

	return opts
}

func wrappedFlagUsages(flags *pflag.FlagSet) string {
	w, _, err := term.GetSize(0)
	if err != nil {
		w = 80
	}

	return flags.FlagUsagesWrapped(w)
}
