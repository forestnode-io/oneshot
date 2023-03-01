export function sendString(string: string): Promise<Response> {
    if (string.length === 0) {
        return Promise.reject(new Error("cannot send empty data"));
    }

    return fetch("/", {
        method: "POST",
        headers: {
            "Content-Length": string.length.toString(),
        },
        body: string,
    })
}