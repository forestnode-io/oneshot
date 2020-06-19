package cmd

import (
	"bufio"
	"github.com/raphaelreyna/oneshot/internal/handlers"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func Execute() {
	SetFlags()
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	rand.Seed(time.Now().UTC().UnixNano())
	err := setupUsernamePassword()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// Determine which mode user wants oneshot to run in
	mode = downloadMode
	if cgi || cgiStrict || shellCommand {
		mode = cgiMode
	}

	srvr := server.NewServer()
	srvr.Done = make(chan map[*server.Route]error)
	srvr.Port = port
	srvr.CertFile = certFile
	srvr.KeyFile = keyFile

	if !noInfo && !noError {
		srvr.InfoLog = log.New(os.Stdout, "\n", 0)
	}
	if !noError {
		srvr.ErrorLog = log.New(os.Stderr, "\nerror :: ", log.LstdFlags)
	}

	var route *server.Route
	switch mode {
	case downloadMode:
		route = downloadSetup(cmd, args, srvr)
	case cgiMode:
		route, err = cgiSetup(cmd, args, srvr)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}

	if password != "" || username != "" {
		route.HandlerFunc = handlers.Authenticate(username, password,
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
			}, route.HandlerFunc)
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
					os.Exit(0)
				}()
			} else {
				srvr.Close()
				os.Exit(0)
			}
			sigCount++
		}
	}()

	// Handle timer if user gave timeout duration
	if timeout != 0 {
		_ = time.AfterFunc(timeout, func() {
			srvr.Shutdown(cmd.Context())
			os.Exit(0)
		})
	}

	go srvr.Serve()
	<-srvr.Done
	srvr.Shutdown(cmd.Context())
	os.Exit(0)
}

func setupUsernamePassword() error {
	if passwordHidden {
		os.Stdout.WriteString("password: ")
		passreader := bufio.NewReader(os.Stdin)
		passwordBytes, err := passreader.ReadString('\n')
		if err != nil {
			return err
		}
		password = string(passwordBytes)
		password = strings.TrimSpace(password)
		os.Stdout.WriteString("\n")
	} else if passwordFile != "" {
		passwordBytes, err := ioutil.ReadFile(passwordFile)
		if err != nil {
			return err
		}
		password = string(passwordBytes)
		password = strings.TrimSpace(password)
	}
	return nil
}
