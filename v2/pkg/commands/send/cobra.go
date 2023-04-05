package send

import (
	"fmt"
	"mime"
	"path/filepath"

	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/raphaelreyna/oneshot/v2/pkg/commands"
	"github.com/raphaelreyna/oneshot/v2/pkg/configuration"
	"github.com/raphaelreyna/oneshot/v2/pkg/file"
	"github.com/raphaelreyna/oneshot/v2/pkg/output"
	"github.com/spf13/cobra"
)

func New(config *configuration.Root) *Cmd {
	return &Cmd{
		config: config,
	}
}

type Cmd struct {
	rtc          file.ReadTransferConfig
	cobraCommand *cobra.Command

	config *configuration.Root
}

func (c *Cmd) Cobra() *cobra.Command {
	if c.cobraCommand != nil {
		return c.cobraCommand
	}

	c.cobraCommand = &cobra.Command{
		Use:   "send [file|dir]",
		Short: "Send a file or directory to the client",
		Long: `Send a file or directory to the client. If no file or directory is given, stdin will be used.
When sending from stdin, requests are blocked until an EOF is received; content from stdin is buffered for subsequent requests.
If a directory is given, it will be archived and sent to the client; oneshot does not support sending unarchived directories.
`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			config := c.config.Subcommands.Send
			config.MergeFlags()
			if err := config.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}
			if err := config.Hydrate(); err != nil {
				return fmt.Errorf("failed to hydrate configuration: %w", err)
			}
			return nil
		},
		RunE: c.setHandlerFunc,
	}

	c.config.Subcommands.Send.SetFlags(c.cobraCommand, c.cobraCommand.Flags())

	return c.cobraCommand
}

func (c *Cmd) setHandlerFunc(cmd *cobra.Command, args []string) error {
	var (
		ctx   = cmd.Context()
		paths = args

		config        = c.config.Subcommands.Send
		fileName      = config.Name
		fileMime      = config.MIME
		archiveMethod = string(config.ArchiveMethod)
	)

	output.IncludeBody(ctx)

	if len(paths) == 1 && fileName == "" {
		fileName = filepath.Base(paths[0])
	}

	if fileName != "" && fileMime == "" {
		ext := filepath.Ext(fileName)
		fileMime = mime.TypeByExtension(ext)
	}

	if fileName == "" {
		fileName = namesgenerator.GetRandomName(0)
	}

	var err error
	c.rtc, err = file.NewReadTransferConfig(archiveMethod, args...)
	if err != nil {
		return err
	}

	if file.IsArchive(c.rtc) {
		fileName += "." + archiveMethod
	}

	if _, ok := config.Header["Content-Type"]; !ok {
		config.Header["Content-Type"] = []string{fileMime}
	}
	// Are we triggering a file download on the users browser?
	if !config.NoDownload {
		if _, ok := config.Header["Content-Disposition"]; !ok {
			config.Header["Content-Disposition"] = []string{
				fmt.Sprintf("attachment;filename=%s", fileName),
			}
		}
	}

	commands.SetHTTPHandlerFunc(ctx, c.ServeHTTP)
	return nil
}
