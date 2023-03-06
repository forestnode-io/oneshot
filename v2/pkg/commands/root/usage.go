package root

const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if and .HasAvailableLocalFlags (ne .Name "oneshot")}}

Flags:
{{.LocalFlags | wrappedFlagUsages | trimTrailingWhitespaces}}{{end}}

Global Flags:
{{ "Output flags:" | indent 4}}
{{flags . | wrappedFlagUsages | trimTrailingWhitespaces | indent 8}}

{{ "Server Flags:" | indent 4 }}
{{serverFlags . | wrappedFlagUsages | trimTrailingWhitespaces | indent 8}}

{{ "Basic Authentication Flags:" | indent 4 }}
{{basicAuthFlags . | wrappedFlagUsages | trimTrailingWhitespaces | indent 8}}

{{ "CORS Flags:" | indent 4 }}
{{corsFlags . | wrappedFlagUsages | trimTrailingWhitespaces | indent 8}}

{{ "NAT Traversal Flags:" | indent 4 }}
{{ "WebRTC Flags:" | indent 8 }}
{{webrtcFlags . | wrappedFlagUsages | trimTrailingWhitespaces | indent 12}}{{if eq .Name "oneshot" }}

Use "oneshot [command] --help" for more information about a command.{{end}}
`
