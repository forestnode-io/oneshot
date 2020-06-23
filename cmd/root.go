package cmd

import (
	"bufio"
	"fmt"
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
	port = strings.ReplaceAll(port, ":", "")

	// Determine which mode user wants oneshot to run in
	mode = downloadMode
	if upload {
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
	case uploadMode:
		route, err = uploadSetup(cmd, args, srvr)
		if err != nil {
			log.Println(err)
			os.Exit(1)
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
			if upload && len(args) == 0 && dir == "" && fileName == "" {
				// oneshot will only print received file to stdout
				// print to stderr or a file instead
				if srvr.ErrorLog == nil {
					f, err := os.Create("./oneshot-credentials.txt")
					if err != nil {
						log.Println(err)
						f.Close()
						os.Exit(1)
					}
					msg += "\n" + time.Now().Format("15:04:05.000 MST 2 Jan 2006")
					_, err = f.WriteString(msg)
					if err != nil {
						f.Close()
						log.Println(err)
						os.Exit(1)
					}
					f.Close()
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

func randomPassword() string {
	const lowerChars = "abcdefghijklmnopqrstuvwxyz"
	const upperChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const numericChars = "1234567890"

	var defSeperator = "-"

	runes := []rune(lowerChars + upperChars + numericChars)
	l := len(runes)
	password := ""
	for i := 1; i < 15; i++ {
		if i%5 == 0 {
			password += defSeperator
			continue
		}
		password += string(runes[rand.Intn(l)])
	}
	return password
}

func randomUsername() string {
	adjs := [...]string{"bulky", "fake", "artistic", "plush", "ornate", "kind", "nutty", "miniature", "huge", "evergreen", "several", "writhing", "scary", "equatorial", "obvious", "rich", "beneficial", "actual", "comfortable", "well-lit"}

	nouns := [...]string{"representative", "prompt", "respond", "safety", "blood", "fault", "lady", "routine", "position", "friend", "uncle", "savings", "ambition", "advice", "responsibility", "consist", "nobody", "film", "attitude", "heart"}

	l := len(adjs)

	return adjs[rand.Intn(l)] + "_" + nouns[rand.Intn(l)]
}
