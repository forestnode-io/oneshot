package root

const helpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}
If you encounter any bugs or have any questions or suggestions, please open an issue at:
https://github.com/forestnode-io/oneshot/issues/new/choose
`
