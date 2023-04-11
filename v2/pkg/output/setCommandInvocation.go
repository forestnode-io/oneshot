package output

import (
	"github.com/spf13/cobra"
)

func (o *output) setCommandInvocation(cmd *cobra.Command, args []string) {
	var (
		name = cmd.Name()
		argc = len(args)

		includeContent = func() {
			// if outputting json report and executing a command, include the sent body in the report
			// since the user may not have a copy of it laying around
			if _, exclude := o.FormatOpts["exclude-file-contents"]; !exclude {
				o.FormatOpts["include-file-contents"] = struct{}{}
			}
		}
	)

	o.gotInvocationInfo = true
	o.cmdName = name
	cmd.VisitParents(func(c *cobra.Command) {
		if c.Name() == "oneshot" {
			return
		}
		o.cmdName = c.Name() + " " + o.cmdName
	})

	switch o.cmdName {
	case "exec":
		if o.Format == "json" {
			includeContent()
		}
	case "redirect":
	case "webrtc client send":
		fallthrough
	case "send":
		switch argc {
		case 0: // sending from stdin
			if o.Format != "json" {
				o.enableDynamicOutput()
			} else {
				includeContent()
			}
		default: // sending file(s)
			if o.Format != "json" {
				o.enableDynamicOutput()
			}
		}
	case "webrtc client receive":
		fallthrough
	case "receive":
		if !o.quiet {
			o.enableDynamicOutput()
		}
		switch argc {
		case 0: // receiving to stdout
			if o.Format == "json" {
				includeContent()
			}
		default: // receiving to a file
		}
	case "reverse-proxy":
		if o.Format == "json" {
			includeContent()
		}
	case "webrtc signalling-server":
	case "webrtc browser-client":
	case "version":
	default:
	}
}
