package cmd

import (
	"github.com/spf13/cobra"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"os"
	"time"
	"os/signal"
	"syscall"
	"log"
)

var (
	noInfo bool
	noError bool
	port string
	timeout time.Duration
)

var RootCmd = &cobra.Command{
	Use: "oneshot [flags]... FILE",
	Short: "A single-fire HTTP server.",
	Long: "Start an HTTP server which will only serve one file once before exiting.",
	Run: run,
}

func Execute() {
	RootCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to bind to.")
	RootCmd.Flags().DurationVarP(&timeout, "timeout", "t", 0,
		`How long to wait for client.
A value of zero will set the timeout to the max possible value.`,
	)
	RootCmd.Flags().BoolVarP(&noInfo, "quiet", "q", false, "Nothing will be sent to stdout.")
	RootCmd.Flags().BoolVarP(&noInfo, "silent", "Q", false, "Nothing will be sent to stdout and stderr.")

	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(0)
	}
}

func run(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		os.Exit(0)
	}
	srvr := server.NewServer()
	srvr.Done = make(chan struct{})
	srvr.Port = port
	srvr.Timeout = timeout

	srvr.FilePath = args[0]
	if !noInfo && !noError {
		srvr.InfoLog = log.New(os.Stdout, "oneshot :: ", 0)
	}
	if !noError {
		srvr.ErrorLog = log.New(os.Stderr, "oneshot error :: ", log.LstdFlags)
	}

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		srvr.Stop(cmd.Context())
		os.Exit(0)
	}()

	go srvr.Serve(cmd.Context())
	<- srvr.Done
	os.Exit(0)
}
