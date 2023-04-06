package browserclient

const usageTemplate = `Browser client options:
{{ .LocalFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Usage:
  {{ .UseLine }}
`
