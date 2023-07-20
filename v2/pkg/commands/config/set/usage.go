package set

const usageTemplate = `set options:
{{ .LocalFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Usage:
  {{ .UseLine }}
`
