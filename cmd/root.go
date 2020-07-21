package cmd

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/raphaelreyna/oneshot/internal/handlers"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"github.com/spf13/cobra"
)

func Execute() {
	// Allow for execution on Windows by double clicking oneshot.exe
	cobra.MousetrapHelpText = ""

	SetFlags()

	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	returnCode := 1
	defer func() { os.Exit(returnCode) }()

	if msg := os.Getenv("ONESHOT_DONT_EXIT"); msg != "" {
		// If we're asked to not exit then just wait on a channel
		defer func() {
			d := make(chan struct{})

			// Did the user supply a message to display?
			if msg != "T" && msg != "t" {
				os.Stdout.WriteString("\n\n")
				os.Stdout.WriteString(msg)
			}

			// Wait for an empty struct that will never come
			<-d
		}()
	}

	// Clean up the port string
	port = strings.ReplaceAll(port, ":", "")

	rand.Seed(time.Now().UTC().UnixNano())
	err := setupUsernamePassword()
	if err != nil {
		log.Println(err)
		return
	}

	tlsLoc, err := setupCertAndKey(cmd)
	if err != nil {
		log.Println(err)
		return
	}
	if tlsLoc != "" {
		defer os.RemoveAll(tlsLoc)
	}

	// Determine which mode user wants oneshot to run in
	mode = downloadMode
	if upload || uploadFile || uploadInput {
		mode = uploadMode
	}
	if cgi || cgiStrict || shellCommand {
		mode = cgiMode
	}

	// Create the server and start configuring it
	srvr := server.NewServer()
	srvr.Done = make(chan map[*server.Route]error)
	srvr.Port = port
	srvr.CertFile = certFile
	srvr.KeyFile = keyFile

	// Grab all of the machines ip addresses to present to the user
	srvr.HostAddresses, err = getHostIPs(srvr.Port)
	if err != nil {
		log.Println(err)
		return
	}

	// If we are using mdns, the zeroconf server needs to be started up,
	// and the human readable address needs to be prepended to the list of ip addresses.
	if mdns {
		portN, err := strconv.ParseInt(port, 10, 32)
		if err != nil {
			log.Println(err)
			return
		}

		mdnsSrvr, err := zeroconf.Register(
			"oneshot",
			"_http._tcp",
			"local.",
			int(portN),
			[]string{"version=" + version},
			nil,
		)
		defer mdnsSrvr.Shutdown()
		if err != nil {
			log.Println(err)
			return
		}

		host, err := os.Hostname()
		if err != nil {
			log.Println(err)
			return
		}

		srvr.HostAddresses = append([]string{host + ".local" + ":" + port}, srvr.HostAddresses...)
	}

	// Add the loggers to the server based on users preference
	if !noInfo && !noError {
		srvr.InfoLog = log.New(os.Stdout, "", 0)
	}
	if !noError {
		srvr.ErrorLog = log.New(os.Stderr, "error :: ", log.LstdFlags)
	}

	// Create route handler depending on what the user wants to do
	var route *server.Route
	switch mode {
	case downloadMode:
		route, err = downloadSetup(cmd, args, srvr)
		if err != nil {
			log.Println(err)
			return
		}
	case cgiMode:
		route, err = cgiSetup(cmd, args, srvr)
		if err != nil {
			log.Println(err)
			return
		}
	case uploadMode:
		route, err = uploadSetup(cmd, args, srvr)
		if err != nil {
			log.Println(err)
			return
		}
	}

	// Did the user set the username or password flags with empty values?
	// If so, generate a random value
	fs := cmd.Flags()
	var randUser, randPass bool
	if fs.Changed("username") && username == "" {
		username = randomUsername()
		randUser = true
	}
	if fs.Changed("password") && password == "" {
		password = randomPassword()
		randPass = true
	}

	// Are we doing basic web auth?
	if password != "" || username != "" {
		// Wrap the route handler with authentication middle-ware
		route.HandlerFunc = handlers.Authenticate(username, password,
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
			}, route.HandlerFunc)

		// Do we need to show any generated credentials?
		if randPass || randUser {
			msg := ""
			if randUser {
				msg += fmt.Sprintf("generated random username: %s\n", username)
			}
			if randPass {
				msg += fmt.Sprintf("generated random password: %s\n", password)
			}

			// Are we allowed to print to stdout? If not, then what about stderr?
			if (upload && len(args) == 0 && dir == "" && fileName == "") || srvr.InfoLog == nil {
				// oneshot will only print received file to stdout so we print to stderr or a file instead
				if srvr.ErrorLog == nil {
					f, err := os.Create("./oneshot-credentials.txt")
					if err != nil {
						log.Println(err)
						f.Close()
						return
					}
					msg += "\n" + time.Now().Format("15:04:05.000 MST 2 Jan 2006")
					_, err = f.WriteString(msg)
					if err != nil {
						f.Close()
						log.Println(err)
						return
					}
					f.Close()
					defer os.Remove(f.Name())
				} else {
					srvr.ErrorLog.Printf(msg)
				}
			} else {
				srvr.InfoLog.Printf(msg)
			}
		}
	}
	srvr.AddRoute(route)

	// Handle signals from os.
	// User must send signal twice to exit oneshot if data is being transferred
	sigChan := make(chan os.Signal)
	var sigCount int
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for range sigChan {
			if sigCount < 1 {
				go func() {
					srvr.Shutdown(cmd.Context())
					srvr.Done <- nil
				}()
			} else {
				srvr.Close()
				return
			}
			sigCount++
		}
	}()

	// Handle timer if user gave timeout duration
	if timeout != 0 {
		_ = time.AfterFunc(timeout, func() {
			srvr.Shutdown(cmd.Context())
			srvr.Done <- nil
		})
	}

	// Start the server on another goroutine and wait for it to be done
	go srvr.Serve()
	<-srvr.Done
	err = srvr.Shutdown(cmd.Context())
	if err != nil {
		log.Println(err)
		return
	}

	// Everything went okay
	returnCode = 0
}

func getHostIPs(port string) ([]string, error) {
	ips := []string{}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	var home string
	for _, addr := range addrs {
		saddr := addr.String()

		if strings.Contains(saddr, "::") {
			continue
		}

		parts := strings.Split(saddr, "/")
		ip := parts[0] + ":" + port

		// Remove localhost since whats the point in sharing with yourself? (usually)
		if parts[0] == "127.0.0.1" || parts[0] == "localhost" {
			home = ip
			continue
		}

		ips = append(ips, ip)
	}

	if len(ips) == 0 {
		ips = append(ips, home)
	}

	return ips, nil
}
