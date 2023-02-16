export function sendHeader(channel: RTCDataChannel, resource: RequestInfo | URL, options: RequestInit | undefined): void {
    const method = options?.method || 'GET';
    let headerString = `${method} ${resource} HTTP/1.1\n`;

    if (options?.headers) {
        if (!(options?.headers instanceof Headers)) {
            throw new Error("headers must be an instance of Headers");
        }
    };
    const header = options!.headers ? (options!.headers as Headers) : new Headers();
    header.forEach((value, key) => {
        headerString += `${key}: ${value}\n`;
    });
    headerString += '\n';

    channel.send(headerString);
}