package discoveryserver

const usageTemplate = `Output options:
{{ outputFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Server options:
{{ serverFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

Basic Authentication options:
{{ basicAuthFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

CORS options:
{{ corsFlags | wrappedFlagUsages | trimTrailingWhitespaces }}

NAT Traversal options:
{{ "P2P options:" | indent 2 }}
{{ "--p2p-webrtc-config-file string   Path to the configuration file for the underlying WebRTC transport." | indent 4 }}

Usage:
  {{.UseLine}}
`
