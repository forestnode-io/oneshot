package send

const usageTemplate = `Send options:
{{ .LocalFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Output options:
{{ outputFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Server options:
{{ serverFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Basic Authentication options:
{{ basicAuthFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

CORS options:
{{ corsFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

NAT Traversal options:
{{ "P2P options:" | indent 2 }}
{{ p2pFlags | wrappedFlagUsages | trimTrailingWhitespaces | indent 4 }}
{{ "Port mapping options:" | indent 2 }}
{{ upnpFlags | wrappedFlagUsages | trimTrailingWhitespaces | indent 4 }}
{{ "Discovery options:" | indent 2 }}
{{ discoveryServerFlags | wrappedFlagUsages | trimTrailingWhitespaces | indent 4 }}

Usage:
  {{.UseLine}}
`
