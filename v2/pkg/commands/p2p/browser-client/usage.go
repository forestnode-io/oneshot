package browserclient

const usageTemplate = `Browser client options:
{{ .LocalFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Discovery options:
{{ discoveryFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

NAT Traversal options:
{{ "P2P options:" | indent 2 }}
{{ p2pFlags | wrappedFlagUsages | trimTrailingWhitespaces | indent 4 }}

Usage:
  {{ .UseLine }}
`
