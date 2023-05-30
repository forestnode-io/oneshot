package path

const usageTemplate = `path options:
{{ .LocalFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Usage:
  {{ .UseLine }}
`
