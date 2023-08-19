package send

import (
	"fmt"
	"mime"
	"path/filepath"

	"github.com/forestnode-io/oneshot/v2/pkg/commands"
	"github.com/forestnode-io/oneshot/v2/pkg/commands/send/configuration"
	rootconfig "github.com/forestnode-io/oneshot/v2/pkg/configuration"
	"github.com/forestnode-io/oneshot/v2/pkg/file"
	"github.com/forestnode-io/oneshot/v2/pkg/output"
	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/spf13/cobra"
)

func New(config *rootconfig.Root) *Cmd {
	return &Cmd{
		config: config,
	}
}

type Cmd struct {
	rtc          file.ReadTransferConfig
	cobraCommand *cobra.Command

	config *rootconfig.Root
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
		RunE: c.setHandlerFunc,
	}

	c.cobraCommand.SetUsageTemplate(usageTemplate)
	configuration.SetFlags(c.cobraCommand)

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
		return fmt.Errorf("failed to create read transfer config: %w", err)
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
