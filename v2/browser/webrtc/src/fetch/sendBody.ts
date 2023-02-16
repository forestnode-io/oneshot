import { formDataToString } from '../util';

export function sendBody(channel: RTCDataChannel, body: BodyInit | null | undefined): void {
    if (!body) {
        return;
    }

    if (body instanceof FormData) {
        formDataToString(body, "boundary").then((b) => {
            channel.send(b);
        });
    } else if (body instanceof Blob) {
        channel.send(body);
    } else if (body instanceof URLSearchParams) {
        channel.send(body.toString());
    } else if (typeof body === 'string') {
        channel.send(body);
    } else {
        throw new Error(`unknown body type: ${typeof body}`);
    }
}