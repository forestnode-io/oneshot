package receive

const usageTemplate = `Receive options:
{{ .LocalFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Discovery options:
{{ discoveryFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Output options:
{{ outputClientFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Basic Authentication options:
{{ basicAuthClientFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

NAT Traversal options:
{{ "P2P options:" | indent 2 }}
{{ p2pClientFlags | wrappedFlagUsages | trimTrailingWhitespaces | indent 4 }}
`
