package send

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	"github.com/raphaelreyna/oneshot/v2/pkg/file"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/pkg/net/http"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

func New() *Cmd {
	return &Cmd{
		header: make(http.Header),
	}
}

type Cmd struct {
	rtc          file.ReadTransferConfig
	cobraCommand *cobra.Command
	header       http.Header
	status       int
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.header == nil {
		c.header = make(http.Header)
	}
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.header = make(http.Header)
	c.cobraCommand = &cobra.Command{
		Use:   "send [file|dir]",
		Short: "Send a file or directory to the client",
		Long: `Send a file or directory to the client. If no file or directory is given, stdin will be used.
When sending from stdin, requests are blocked until an EOF is received; content from stdin is buffered for subsequent requests.
If a directory is given, it will be archived and sent to the client; oneshot does not support sending unarchived directories.
`,
		RunE: c.setHandlerFunc,
	}

	flags := c.cobraCommand.Flags()
	flags.StringP("archive-method", "a", "tar.gz", `Which archive method to use when sending directories.
Recognized values are "zip" and "tar.gz".`)

	flags.BoolP("no-download", "D", false, "Don't trigger client side browser download.")

	flags.StringP("mime", "m", "", `MIME type of file presented to client.
If not set, either no MIME type or the mime/type of the file will be user, depending on of a file was given.`)

	flags.StringP("name", "n", "", `Name of file presented to client if downloading.
If not set, either a random name or the name of the file will be used, depending on if a file was given.`)

	flags.Int("status-code", 200, "HTTP status code sent to client.")

	return c.cobraCommand
}

func (c *Cmd) setHandlerFunc(cmd *cobra.Command, args []string) error {
	var (
		ctx   = cmd.Context()
		paths = args

		flags            = cmd.Flags()
		headerSlice, _   = flags.GetStringSlice("header")
		fileName, _      = flags.GetString("name")
		fileMime, _      = flags.GetString("mime")
		archiveMethod, _ = flags.GetString("archive-method")
		noDownload, _    = flags.GetBool("no-download")
	)
	output.InvocationInfo(ctx, cmd.Name(), len(args))

	c.status, _ = flags.GetInt("status-code")

	if len(paths) == 1 && fileName == "" {
		fileName = filepath.Base(paths[0])
	}

	if fileName == "" {
		fileName = namesgenerator.GetRandomName(0)
	}

	var err error
	c.rtc, err = file.NewReadTransferConfig(args...)
	if err != nil {
		return err
	}

	// TODO(raphaelreyna): Only accept zip and tar.gz
	if archiveMethod != "zip" && archiveMethod != "tar.gz" {
		archiveMethod = "tar.gz"
	}
	if file.IsArchive(c.rtc) {
		fileName += "." + archiveMethod
	}

	c.header = oneshothttp.HeaderFromStringSlice(headerSlice)
	c.header.Set("Content-Type", fileMime)
	// Are we triggering a file download on the users browser?
	if !noDownload {
		c.header.Set("Content-Disposition",
			fmt.Sprintf("attachment;filename=%s", fileName),
		)
	}

	commands.SetHTTPHandlerFunc(ctx, c.ServeHTTP)
	return nil
}
