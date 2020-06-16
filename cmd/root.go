package cmd

import (
	"bufio"
	"fmt"
	"github.com/raphaelreyna/oneshot/pkg/server"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var version string
var versionFlag bool
var date string

var (
	noInfo     bool
	noError    bool
	port       string
	timeout    time.Duration
	noDownload bool

	fileName string
	fileExt  string
	fileMime string

	certFile string
	keyFile  string

	username       string
	password       string
	passwordFile   string
	passwordHidden bool
)

var RootCmd = &cobra.Command{
	Use:     "oneshot [flags]... [file]",
	Version: fmt.Sprintf(": %s\ndate: %s\nauthor: Raphael Reyna\n", version, date),
	Short:   "A single-fire HTTP server.",
	Long: `Start an HTTP server which will only serve files once.
The first client to connect is given the file, all others receive an HTTP 410 Gone response code.

If no file is given, oneshot will instead serve from stdin and hold the clients connection until receiving the EOF character.
`,
	Run: run,
}

func SetFlags() {
	RootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Version for oneshot.")

	RootCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to bind to.")
	RootCmd.Flags().DurationVarP(&timeout, "timeout", "t", 0,
		`How long to wait for client.
A value of zero will set the timeout to the max possible value.`,
	)
	RootCmd.Flags().BoolVarP(&noInfo, "quiet", "q", false,
		`Don't show info messages.
Use -Q, --silent instead to suppress error messages as well.`,
	)
	RootCmd.Flags().BoolVarP(&noInfo, "silent", "Q", false,
		`Don't show info and error messages.
Use -q, --quiet instead to suppress info messages only.`,
	)
	RootCmd.Flags().BoolVarP(&noDownload, "no-download", "D", false,
		`Don't trigger browser download client side.
If set, the "Content-Disposition" header used to trigger downloads in the clients browser won't be sent.`,
	)
	RootCmd.Flags().StringVarP(&fileName, "name", "n", "",
		`Name of file presented to client.
If not set, either a random name or the name of the file will be used,
depending on if a file was given.`,
	)
	RootCmd.Flags().StringVarP(&fileExt, "ext", "e", "", `Extension of file presented to client.
If not set, either no extension or the extension of the file will be used,
depending on if a file was given.`,
	)
	RootCmd.Flags().StringVarP(&fileMime, "mime", "m", "", `MIME type of file presented to client.
If not set, either no MIME type or the mime/type of the file will be user,
depending on of a file was given.`,
	)

	RootCmd.Flags().StringVar(&certFile, "tls-cert", "", `Certificate file to use for HTTPS.
Key file must also be provided using the --tls-key flag.`,
	)
	RootCmd.Flags().StringVar(&keyFile, "tls-key", "", `Key file to use for HTTPS.
Cert file must also be provided using the --tls-cert flag.`,
	)

	RootCmd.Flags().StringVarP(&username, "username", "U", "", `Username for basic authentication.
If a password is not also provided using either the -P, --password;
-W, --hidden-password; or -w, --password-file flags then the client may enter any password.`,
	)
	RootCmd.Flags().StringVarP(&password, "password", "P", "", `Password for basic authentication.
If a username is not also provided using the -U, --username flag then the client may enter any username.
If either the -W, --hidden-password or -w, --password-file flags are set, this flag will be ignored.`,
	)
	RootCmd.Flags().StringVarP(&passwordFile, "password-file", "w", "", `File containing password for basic authentication.
If a username is not also provided using the -U, --username flag then the client may enter any username.
If the -W, --hidden-password flag is set, this flags will be ignored.`,
	)
	RootCmd.Flags().BoolVarP(&passwordHidden, "hidden-password", "W", false, `Prompt for password for basic authentication.
If a username is not also provided using the -U, --username flag then the client may enter any username.
Takes precedence over the -w, --password-file flag`,
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
	if passwordHidden {
		os.Stdout.WriteString("password: ")
		passreader := bufio.NewReader(os.Stdin)
		passwordBytes, err := passreader.ReadString('\n')
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		password = string(passwordBytes)
		password = strings.TrimSpace(password)
		os.Stdout.WriteString("\n")
	} else if passwordFile != "" {
		passwordBytes, err := ioutil.ReadFile(passwordFile)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		password = string(passwordBytes)
		password = strings.TrimSpace(password)
	}
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
	srvr := server.NewServer(file)
	srvr.Done = make(chan struct{})
	srvr.Port = port
	srvr.Timeout = timeout
	srvr.Download = !noDownload
	srvr.CertFile = certFile
	srvr.KeyFile = keyFile
	srvr.Username = username
	srvr.Password = password

	if !noInfo && !noError {
		srvr.InfoLog = log.New(os.Stdout, "\n", 0)
		file.ProgressWriter = os.Stdout
	}
	if !noError {
		srvr.ErrorLog = log.New(os.Stderr, "\nerror :: ", log.LstdFlags)
	}

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		srvr.Close()
		os.Exit(0)
	}()

	go srvr.Serve(cmd.Context())
	<-srvr.Done
	os.Exit(0)
}
