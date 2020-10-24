package cmd

import (
	"fmt"
	"github.com/raphaelreyna/oneshot/cmd/conf"
	"github.com/raphaelreyna/oneshot/internal/server"
	"github.com/spf13/cobra"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var version string
var date string

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
		Use:     "oneshot [flags]... [file|dir|url]",
		Version: fmt.Sprintf(": %s\nbuild date : %s\nauthor : Raphael Reyna", version, date),
		Short:   "A single-fire first-come-first-serve HTTP server.",
		Long: `Transfer files and data easily between your computer and any browser or HTTP client.
The first client to connect is given the file or uploads a file, all others receive an HTTP 410 Gone response code.
Directories will automatically be archived before being sent (see -a, --archive-method for more information).
`,
		Run: app.Run,
	}

	// Create configuration
	app.conf = conf.NewConf(app.cmd)

	return app, nil
}

func (a *App) SetFlags() {
	a.conf.SetFlags(a.cmd)
}

func (a *App) Cmd() *cobra.Command {
	return a.cmd
}

func (a *App) Start() {
	cobra.MousetrapHelpText = ""

	a.SetFlags()

	if err := a.cmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func (a *App) handleSignal(srvr *server.Server, sigChan chan os.Signal, c chan struct{}) {
	// User must send signal twice to exit oneshot if data is being transferred
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	var sigCount int
	go func() {
		for range sigChan {
			if sigCount < 1 {
				c <- struct{}{}
			} else {
				srvr.Close()
				close(sigChan)
				return
			}
			sigCount++
		}
	}()
}

func (a *App) blockExit(msg string) {
	d := make(chan struct{})

	// Did the user supply a message to display?
	if msg != "T" && msg != "t" {
		os.Stdout.WriteString("\n\n")
		os.Stdout.WriteString(msg)
	}

	// Wait for an empty struct that will never come
	<-d
}
