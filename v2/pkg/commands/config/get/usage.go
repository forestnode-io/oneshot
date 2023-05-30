package get

const usageTemplate = `get options:
{{ .LocalFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Usage:
  {{ .UseLine }}
`
