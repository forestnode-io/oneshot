package send

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"

	"github.com/raphaelreyna/oneshot/v2/internal/commands"
	"github.com/raphaelreyna/oneshot/v2/internal/file"
	oneshothttp "github.com/raphaelreyna/oneshot/v2/internal/net/http"
	"github.com/spf13/cobra"
)

func New() *Cmd {
	return &Cmd{
		header: make(http.Header),
	}
}

type Cmd struct {
	file   *file.FileReader
	cmd    *cobra.Command
	header http.Header
}

func (c *Cmd) Cobra() *cobra.Command {
	c.header = make(http.Header)
	c.cmd = &cobra.Command{
		Use:   "send [file|dir]",
		Short: "",
		Long:  "",
		RunE:  c.createServer,
	}

	flags := c.cmd.Flags()
	flags.StringP("archive-method", "a", "tar.gz", "Which archive method to use when sending directories.\nRecognized values are \"zip\" and \"tar.gz\".")
	flags.BoolP("stream", "J", false, "Stream contents when sending stdin, don't wait for EOF.")
	flags.BoolP("no-download", "D", false, "Don't trigger client side browser download.")
	flags.StringP("extension", "e", "", "Extension of file presented to client.\nIf not set, either no extension or the extension of the file will be used, depending on if a file was given.")
	flags.StringP("mime", "m", "", "MIME type of file presented to client.\nIf not set, either no MIME type or the mime/type of the file will be user, depending on of a file was given.")
	flags.StringP("name", "n", "", "Name of file presented to client if downloading.\nIf not set, either a random name or the name of the file will be used,depending on if a file was given.")
	flags.Int("status-code", 200, "HTTP status code sent to client.")

	return c.cmd
}

func (c *Cmd) createServer(cmd *cobra.Command, args []string) error {
	var (
		ctx   = cmd.Context()
		paths = args

		flags            = cmd.Flags()
		headerSlice, _   = flags.GetStringSlice("header")
		fileName, _      = flags.GetString("name")
		fileExt, _       = flags.GetString("ext")
		fileMime, _      = flags.GetString("mime")
		archiveMethod, _ = flags.GetString("archive-method")
		stream, _        = flags.GetBool("stream")
	)

	if len(paths) == 1 && fileName == "" {
		fileName = filepath.Base(paths[0])
	}

	// TODO(raphaelreyna): Only accept zip and tar.gz
	if archiveMethod != "zip" && archiveMethod != "tar.gz" {
		archiveMethod = "tar.gz"
	}

	if len(paths) == 0 {
		// serving from stdin
		if !stream {
			// dont serve http until stdin stream hits EOF
			tdir, err := os.MkdirTemp("", "oneshot")
			if err != nil {
				return err
			}

			if fileName == "" {
				fileName = fmt.Sprintf("%0-x", rand.Int31())
			}

			fp := filepath.Join(tdir, fileName, fileExt)
			paths = append(paths, fp)

			err = func() error {
				tfile, err := os.Create(fp)
				if err != nil {
					return err
				}
				defer tfile.Close()

				_, err = io.Copy(tfile, os.Stdin)
				return err
			}()
			if err != nil {
				return err
			}
		}
	} else {
		for _, path := range paths {
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("unable to read file: %s", path)
			}
		}
	}

	c.header = oneshothttp.HeaderFromStringSlice(headerSlice)
	c.file = &file.FileReader{
		Paths:         paths,
		Name:          fileName,
		Ext:           fileExt,
		MimeType:      fileMime,
		ArchiveMethod: archiveMethod,
	}

	commands.SetHTTPHandlerFunc(ctx, c.ServeHTTP)
	return nil
}
