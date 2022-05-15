package commands

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"

	"github.com/raphaelreyna/oneshot/v2/internal/file"
	"github.com/raphaelreyna/oneshot/v2/internal/server"
	"github.com/spf13/cobra"
)

func init() {
	d := sendCmd{
		header: make(http.Header),
	}
	root.AddCommand(d.command())
}

type sendCmd struct {
	file   *file.FileReader
	cmd    *cobra.Command
	header http.Header
}

func (s *sendCmd) command() *cobra.Command {
	s.cmd = &cobra.Command{
		Use:   "send [file|dir]",
		Short: "",
		Long:  "",
		RunE:  s.runE,
	}

	pflags := s.cmd.PersistentFlags()

	pflags.BoolP("allow-bots", "B", false, "Allow bots to attempt transfer.")
	pflags.StringP("extension", "e", "", "Extension of file presented to client.\nIf not set, either no extension or the extension of the file will be used, depending on if a file was given.")
	pflags.StringArrayP("header", "H", nil, "HTTP header to send to client.\nSetting a value for 'Content-Type' will override the -M, --mime flag.")
	pflags.StringP("mime", "m", "", "MIME type of file presented to client.\nIf not set, either no MIME type or the mime/type of the file will be user, depending on of a file was given.")
	pflags.StringP("name", "n", "", "Name of file presented to client or if uploading, the name of the file saved to this computer.\nIf not set, either a random name or the name of the file will be used,depending on if a file was given.")
	pflags.BoolP("no-download", "D", false, "Don't trigger client side browser download.")

	lflags := s.cmd.Flags()
	lflags.StringP("archive-method", "a", "tar.gz", "Which archive method to use when sending directories.\nRecognized values are \"zip\" and \"tar.gz\".")
	lflags.BoolP("stream", "J", false, "Stream contents when sending stdin, don't wait for EOF.")

	return s.cmd
}

func (s *sendCmd) runE(cmd *cobra.Command, args []string) error {
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
			tdir, err := ioutil.TempDir("", "oneshot")
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
	}

	s.header = headerFromStringSlice(headerSlice)
	s.file = &file.FileReader{
		Paths:         paths,
		Name:          fileName,
		Ext:           fileExt,
		MimeType:      fileMime,
		ArchiveMethod: archiveMethod,
	}

	srvr := server.NewServer(s)
	setServer(ctx, srvr)
	return nil
}

func (s *sendCmd) ServeHTTP(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var (
		file = s.file

		cmd           = s.cmd
		flags         = cmd.Flags()
		noDownload, _ = flags.GetBool("no-download")
		allowBots, _  = flags.GetBool("allow-bots")

		header = s.header
	)

	// Filter out requests from bots, iMessage, etc. by checking the User-Agent header for known bot headers
	if headers, exists := r.Header["User-Agent"]; exists && !allowBots {
		if isBot(headers) {
			w.WriteHeader(http.StatusOK)
			return struct{}{}, errors.New("bot")
		}
	}

	err := file.Open()
	defer func() {
		file.Reset()
	}()
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return struct{}{}, err
	}

	// Are we triggering a file download on the users browser?
	if !noDownload {
		w.Header().Set("Content-Disposition",
			fmt.Sprintf("attachment;filename=%s", file.Name),
		)
	}

	// Set standard Content headers
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", file.Size()))

	// Set any additional headers added by the user via flags
	for key := range header {
		w.Header().Set(key, header.Get(key))
	}

	// Start writing the file data to the client while timing how long it takes
	_, err = io.Copy(w, file)
	if err != nil {
		return struct{}{}, err
	}

	return struct{}{}, nil
}

func (d *sendCmd) ServeExpiredHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("expired hello from server"))
}
