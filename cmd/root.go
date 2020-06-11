package cmd

import (
	"github.com/spf13/cobra"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"os"
	"log"
)

var RootCmd = &cobra.Command{
	Use: "oneshot",
	Short: "A single-fire HTTP server.",
	Long: "oneshot starts an HTTP server which will only serve one file once before exiting.",
	Run: oneshot,
}

var srvr *server.Server

var (
	noInfo bool
	noError bool
)

func init() {
	srvr = server.NewServer()
	RootCmd.Flags().StringVarP(&srvr.Port, "port", "p", "8080", "Port to bind to.")
	RootCmd.Flags().DurationVarP(&srvr.Timeout, "timeout", "t", 0,
		"How long to wait for client. A value of zero will set the timeout to the max possible value.",
	)
	RootCmd.Flags().BoolVarP(&noInfo, "quiet", "q", false, "Nothing will be sent to stdout.")
	RootCmd.Flags().BoolVarP(&noInfo, "silent", "Q", false, "Nothing will be sent to stdout and stderr.")
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(0)
	}
}

func oneshot(cmd *cobra.Command, args []string) {
	srvr.Done = make(chan struct{})
	srvr.FilePath = args[0]
	if !noInfo && !noError {
		srvr.InfoLog = log.New(os.Stdout, "oneshot :: ", 0)
	}
	if !noError {
		srvr.ErrorLog = log.New(os.Stderr, "oneshot error :: ", log.LstdFlags)
	}
	go srvr.Serve()
	<- srvr.Done
	os.Exit(0)
}
