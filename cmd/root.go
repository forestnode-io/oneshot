package cmd

import (
	"bufio"
	"github.com/raphaelreyna/oneshot/internal/handlers"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func Execute() {
	SetFlags()
	err := setupUsernamePassword()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(0)
	}
}

func run(cmd *cobra.Command, args []string) {
	srvr := server.NewServer()
	srvr.Done = make(chan map[*server.Route]error)
	srvr.Port = port
	srvr.CertFile = certFile
	srvr.KeyFile = keyFile

	var filePath string
	if len(args) >= 1 {
		filePath = args[0]
	}
	if filePath != "" && fileName != "" {
		fileName = filepath.Base(filePath)
	}
	file := &server.File{
		Path:     filePath,
		Name:     fileName,
		Ext:      fileExt,
		MimeType: fileMime,
	}

	if !noInfo && !noError {
		srvr.InfoLog = log.New(os.Stdout, "\n", 0)
		file.ProgressWriter = os.Stdout
	}
	if !noError {
		srvr.ErrorLog = log.New(os.Stderr, "\nerror :: ", log.LstdFlags)
	}

	dwnldRoute := downloadRoute()
	dwnldRoute.HandlerFunc = handlers.HandleSend(file, !noDownload, srvr.InfoLog)

	if password != "" || username != "" {
		dwnldRoute.HandlerFunc = handlers.Authenticate(username, password,
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
			}, dwnldRoute.HandlerFunc)
	}

	srvr.AddRoute(dwnldRoute)

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
	err := (<-srvr.Done)[dwnldRoute]
	if err != server.OKDoneErr {
		if !noError {
			srvr.ErrorLog.Println(err)
		}
	}
	srvr.Shutdown(cmd.Context())
	os.Exit(0)
}

func downloadRoute() *server.Route {
	r := &server.Route{
		Pattern: "/",
		Methods: []string{"GET"},
		MaxOK:   1,
		DoneHandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusGone)
			w.Write([]byte("gone"))
		},
	}
	return r
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
