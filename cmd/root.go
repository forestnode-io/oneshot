package cmd

import (
	"github.com/raphaelreyna/oneshot/pkg/server"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	noInfo  bool
	noError bool
	port    string
	timeout time.Duration

	fileName string
	fileExt  string
	fileMime string
)

var RootCmd = &cobra.Command{
	Use:   "oneshot [flags]... [file]",
	Short: "A single-fire HTTP server.",
	Long: `Start an HTTP server which will only serve files once.
If no file is given, oneshot will instead serve from stdin.
If serving from stdin, oneshot will hold the clients connection until receiving the EOF character`,
	Run: run,
}

func SetFlags() {
	RootCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to bind to.")
	RootCmd.Flags().DurationVarP(&timeout, "timeout", "t", 0,
		`How long to wait for client.
A value of zero will set the timeout to the max possible value.`,
	)
	RootCmd.Flags().BoolVarP(&noInfo, "quiet", "q", false,
		`Don't show info messages.
Use -Q instead to suppress error messages as well.`,
	)
	RootCmd.Flags().BoolVarP(&noInfo, "silent", "Q", false,
		`Don't show info and error messages.
Use -q instead to suppress info messages only.`,
	)
	RootCmd.Flags().StringVarP(&fileName, "name", "n", "",
		`Name of file presented to client.
If not set, either a random name or the name of the file will be used, depending on if a file was given.`,
	)
	RootCmd.Flags().StringVarP(&fileExt, "ext", "e", "", `Extension of file presented to client.
If not set, either no extension or the extension of the file will be used, depending on if a file was given.`,
	)
	RootCmd.Flags().StringVarP(&fileMime, "mime", "m", "", `MIME type of file presented to client.
If not set, either no MIME type or the mime/type of the file will be user, depending on of a file was given.`,
	)
}

func Execute() {
	SetFlags()
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(0)
	}
}

func run(cmd *cobra.Command, args []string) {
	var filepath string
	if len(args) >= 1 {
		filepath = args[0]
	}
	file := server.File{
		Path:     filepath,
		Name:     fileName,
		Ext:      fileExt,
		MimeType: fileMime,
	}
	srvr := server.NewServer(&file)
	srvr.Done = make(chan struct{})
	srvr.Port = port
	srvr.Timeout = timeout

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
	<-srvr.Done
	os.Exit(0)
}
