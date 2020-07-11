package cmd

import (
	"fmt"
	"log"
	"math/rand"
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
	cobra.MousetrapHelpText = ""
	SetFlags()
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	returnCode := 0
	defer func() { os.Exit(returnCode) }()

	if msg := os.Getenv("ONESHOT_DONT_EXIT"); msg != "" {
		defer func() {
			d := make(chan struct{})
			if msg != "T" && msg != "t" {
				os.Stdout.WriteString("\n\n")
				os.Stdout.WriteString(msg)
			}
			<-d
		}()
	}

	port = strings.ReplaceAll(port, ":", "")

	rand.Seed(time.Now().UTC().UnixNano())
	err := setupUsernamePassword()
	if err != nil {
		log.Println(err)
		returnCode = 1
		return
	}

	tlsLoc, err := setupCertAndKey(cmd)
	if err != nil {
		log.Println(err)
		returnCode = 1
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

	srvr := server.NewServer()
	srvr.Done = make(chan map[*server.Route]error)
	srvr.Port = port
	srvr.CertFile = certFile
	srvr.KeyFile = keyFile

	if mdns {
		portN, err := strconv.ParseInt(port, 10, 32)
		if err != nil {
			log.Println(err)
			returnCode = 1
			return
		}
		if err != nil {
			log.Println(err)
			returnCode = 1
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

		host, err := os.Hostname()
		if err != nil {
			log.Println(err)
			returnCode = 1
			return
		}
		if certFile != "" && keyFile != "" {
			srvr.MDNSAddress = "https://"
		} else {
			srvr.MDNSAddress = "http://"
		}
		srvr.MDNSAddress += host + ".local" + ":" + port
	}

	if !noInfo && !noError {
		srvr.InfoLog = log.New(os.Stdout, "", 0)
	}
	if !noError {
		srvr.ErrorLog = log.New(os.Stderr, "error :: ", log.LstdFlags)
	}

	var route *server.Route
	switch mode {
	case downloadMode:
		route = downloadSetup(cmd, args, srvr)
	case cgiMode:
		route, err = cgiSetup(cmd, args, srvr)
		if err != nil {
			log.Println(err)
			returnCode = 1
			return
		}
	case uploadMode:
		route, err = uploadSetup(cmd, args, srvr)
		if err != nil {
			log.Println(err)
			returnCode = 1
			return
		}
	}

	fs := cmd.Flags()
	var randUser bool
	var randPass bool
	if fs.Changed("username") && username == "" {
		username = randomUsername()
		randUser = true
	}
	if fs.Changed("password") && password == "" {
		password = randomPassword()
		randPass = true
	}

	if password != "" || username != "" {
		route.HandlerFunc = handlers.Authenticate(username, password,
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
			}, route.HandlerFunc)
		if randPass || randUser {
			msg := ""
			if randUser {
				msg += fmt.Sprintf("generated random username: %s\n", username)
			}
			if randPass {
				msg += fmt.Sprintf("generated random password: %s\n", password)
			}
			// Are we allowed to print to stdout?
			if (upload && len(args) == 0 && dir == "" && fileName == "") || srvr.InfoLog == nil {
				// oneshot will only print received file to stdout
				// print to stderr or a file instead
				if srvr.ErrorLog == nil {
					f, err := os.Create("./oneshot-credentials.txt")
					if err != nil {
						log.Println(err)
						f.Close()

						returnCode = 1
						return
					}
					msg += "\n" + time.Now().Format("15:04:05.000 MST 2 Jan 2006")
					_, err = f.WriteString(msg)
					if err != nil {
						f.Close()
						log.Println(err)

						returnCode = 1
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
		for _ = range sigChan {
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

	go srvr.Serve()
	<-srvr.Done
	srvr.Shutdown(cmd.Context())
}
