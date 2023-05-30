package config

const usageTemplate = `Usage:
	{{ .CommandPath }} [command]

Available Commands: {{ range .Commands }}{{if (or .IsAvailableCommand (eq .Name "help"))}}
	{{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}
`
