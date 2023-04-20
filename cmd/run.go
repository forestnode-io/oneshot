package cmd

import (
	"log"
	"os"
	"time"

	"github.com/oneshot-uno/oneshot/internal/server"
	"github.com/spf13/cobra"
	"io"
	"path/filepath"
	"strings"
)

const stdinBufferSize = 1024

func (a *App) Run(cmd *cobra.Command, args []string) {
	returnCode := 1
	defer func() { os.Exit(returnCode) }()

	// If we're asked to not exit then block
	if msg := os.Getenv("ONESHOT_DONT_EXIT"); msg != "" {
		defer a.blockExit(msg)
	}

	// Parse and prepare configuration
	c := a.conf
	err := c.Parse()
	if err != nil {
		log.Println(err)
		return
	}

	// Clean up if using self signed tls cert
	if loc, exists := c.SSTLSLoc(); exists {
		defer os.RemoveAll(loc)
	}
	// Clean up if randomly generated credentials were save to disk
	if loc, exists := c.CredFileLoc(); exists {
		defer os.Remove(loc)
	}

	// Create the server and configure it
	srvr := server.NewServer()
	srvr.Done = make(chan map[*server.Route]error)
	err = c.SetupServer(srvr, args, a.ips)
	if err != nil {
		log.Println(err)
		return
	}

	// Handle mDNS
	err = a.MDNS(version, srvr)
	if err != nil && srvr.ErrorLog != nil {
		srvr.ErrorLog.Println(err)
		return
	}

	// Handle timer if user gave timeout duration
	var timer <-chan time.Time
	if c.Timeout != 0 {
		timer = time.After(c.Timeout)
	}

	// Handle signals from os.
	shouldExitChan := make(chan struct{})
	a.handleSignal(srvr, make(chan os.Signal), shouldExitChan)

	// Should we wait until we get an EOF on stdin to start serving HTTP?
	if loc := c.StdinBufferLocation(); loc != "" {
		sf, err := os.Create(loc)

		// Clean up on exit
		defer os.RemoveAll(filepath.Dir(loc))
		if err != nil {
			log.Println(err)
			return
		}

		// Read in from stdin into temp file
		si := os.Stdin
		b := make([]byte, stdinBufferSize)
		for n, e := si.Read(b); e == nil; n, e = si.Read(b) {
			sf.Write(b[0:n])
		}
		if err != nil && err != io.EOF {
			log.Println(err)
			return
		}

		err = sf.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}

	// Start the HTTP(S) server
	go func() {
		err := srvr.Serve()
		// Check to see if the port is already running, exit if so
		if err != nil {
			// Filter out error message that comes up when user disconnects after using HTTPS
			if srvr.ErrorLog != nil && !strings.Contains(err.Error(), "Server closed") {
				srvr.ErrorLog.Println(err)
			}
			shouldExitChan <- struct{}{}
		}
	}()

	// Wait for either the server to be done, the time to expire, or something to go wrong
	select {
	case <-shouldExitChan:
		returnCode = 1
	case <-timer:
		returnCode = 0
	case <-srvr.Done:
		returnCode = 0
	}

	// Gracefully shutdown server
	err = srvr.Shutdown(cmd.Context())
	if err != nil {
		returnCode = 1
		log.Println(err)
		return
	}
}
