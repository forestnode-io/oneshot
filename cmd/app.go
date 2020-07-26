package cmd

import (
	"fmt"
	"github.com/raphaelreyna/oneshot/cmd/conf"
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
	conf     *conf.Conf

	server *server.Server
}

// Create an unconfigured app instance
func NewApp() (*App, error) {
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
		Run: app.Run,
	}

	// Create configuration
	app.conf = conf.NewConf()

	return app, nil
}

func (a *App) Start() {
	cobra.MousetrapHelpText = ""

	a.conf.SetFlags(a.cmd)

	if err := a.cmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
