package conf

import (
	"fmt"
	"github.com/raphaelreyna/oneshot/internal/handlers"
	"github.com/raphaelreyna/oneshot/internal/server"
	"log"
	"net/http"
	"os"
	"time"
)

func (c *Conf) SetupServer(srvr *server.Server, args []string, ips []string) error {
	var err error

	srvr.Port = c.Port
	srvr.CertFile = c.CertFile
	srvr.KeyFile = c.KeyFile

	// Set reachable addresses and append port
	srvr.HostAddresses = []string{}
	for _, ip := range ips {
		srvr.HostAddresses = append(srvr.HostAddresses, ip+":"+c.Port)
	}

	// Add the loggers to the server based on users preference
	if !c.NoInfo && !c.NoError {
		srvr.InfoLog = log.New(os.Stdout, "", 0)
	}
	if !c.NoError {
		srvr.ErrorLog = log.New(os.Stderr, "error :: ", log.LstdFlags|log.Lshortfile)
	}

	// Create route handler depending on what the user wants to do
	var route *server.Route
	switch c.Mode() {
	case DownloadMode:
		route, err = c.setupDownloadRoute(args, srvr)
	case CGIMode:
		route, err = c.setupCGIRoute(args, srvr)
	case UploadMode:
		route, err = c.setupUploadRoute(args, srvr)
	}
	if err != nil {
		return err
	}

	// Are we doing basic web auth?
	if c.Password != "" || c.Username != "" {
		// Wrap the route handler with authentication middle-ware
		route.HandlerFunc = handlers.Authenticate(c.Username, c.Password,
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
			}, route.HandlerFunc)
	}

	srvr.AddRoute(route)

	// Do we need to show any generated credentials?
	if c.randPass || c.randUser {
		msg := ""
		if c.randUser {
			msg += fmt.Sprintf(
				"generated random username: %s\n",
				c.Username,
			)
		}
		if c.randPass {
			msg += fmt.Sprintf(
				"generated random password: %s\n",
				c.Password,
			)
		}

		// Are we uploading to stdout? If so, we can't print info messages to stdout
		uploadToStdout := c.Upload && len(args) == 0 && c.Dir == "" && c.FileName == ""
		if uploadToStdout || srvr.InfoLog == nil {
			// oneshot will only print received file to stdout so we print to stderr or a file instead
			if srvr.ErrorLog == nil {
				f, err := os.Create("./oneshot-credentials.txt")
				if err != nil {
					f.Close()
					return err
				}
				msg += "\n" + time.Now().Format("15:04:05.000 MST 2 Jan 2006")
				_, err = f.WriteString(msg)
				if err != nil {
					f.Close()
					return err
				}
				f.Close()
				c.credFileLoc = f.Name()
			} else {
				srvr.ErrorLog.Printf(msg)
			}
		} else {
			srvr.InfoLog.Printf(msg)
		}
	}

	return nil
}
