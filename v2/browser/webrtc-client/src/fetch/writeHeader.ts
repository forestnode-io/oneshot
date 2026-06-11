import { DataChannelMTU } from "./constants";

export async function writeHeader(channel: RTCDataChannel, resource: RequestInfo | URL, options?: RequestInit): Promise<void> {
    const method = options?.method || 'GET';
    let headerString = `${method} ${resource} HTTP/1.1\n`;

    let headers = options?.headers ? options!.headers! : new Headers();
    if (headers instanceof Headers) {
        if (!headers.has('User-Agent')) {
            headers.append('User-Agent', navigator.userAgent);
        }
        headers.append("X-HTTPOverWebRTC", "true");
        headers.forEach((value, key) => {
            headerString += `${key}: ${value}\n`;
        });
    } else if (typeof headers === 'object') {
        if (headers instanceof Array) {
            var foundUserAgent = false;
            for (var i = 0; i < headers.length; i++) {
                headerString += `${headers[i][0]}: ${headers[i][1]}\n`;
                if (headers[i][0] === 'User-Agent') {
                    foundUserAgent = true;
                }
            }
            if (!foundUserAgent) {
                headerString += `User-Agent: ${navigator.userAgent}\n`;
            }
            headerString += "X-HTTPOverWebRTC: true\n"
        } else {
            if (!headers['User-Agent']) {
                headers['User-Agent'] = navigator.userAgent;
            }
            headers['X-HTTPOverWebRTC'] = 'true';
            for (const key in headers) {
                headerString += `${key}: ${headers[key]}\n`;
            }
        }
    }
    headerString += '\n';

    console.log("writing header: ", headerString);

    return new Promise<void>((resolve, reject) => {
        const pump = sendPump(channel, headerString, resolve, reject);
        pump();
    });
}

// sendPump sends `data` over the channel in MTU-sized chunks, honoring
// backpressure. It calls resolve() once all data has been flushed and reject()
// on a fatal send error. Because it may pause and resume asynchronously when
// the send buffer fills, callers must wait on the promise before sending more
// data on the same channel to avoid interleaving.
function sendPump(channel: RTCDataChannel, data: string, resolve: () => void, reject: (reason: any) => void): () => void {
    var mtu = DataChannelMTU;
    const s = function () {
        while (data.length) {
            if (channel.bufferedAmount > channel.bufferedAmountLowThreshold) {
                // Buffer is full: pause and resume once it drains below the
                // low threshold. Without returning here we would keep sending
                // and overflow the send buffer (throws on iOS Safari).
                channel.onbufferedamountlow = () => {
                    channel.onbufferedamountlow = null;
                    s();
                }
                return;
            }

            if (data.length < mtu) {
                mtu = data.length;
            }

            const chunk = data.slice(0, mtu);
            data = data.slice(mtu);
            try {
                channel.send(chunk);
            } catch (e) {
                // RTCDataChannels aren't always immediately ready in Safari,
                // even after the open event, so retry once after a short delay.
                if (e instanceof DOMException && e.name === 'InvalidStateError') {
                    setTimeout(() => {
                        try {
                            channel.send(chunk);
                            s();
                        } catch (e) {
                            reject(e);
                        }
                    }, 500);
                    return;
                } else {
                    reject(e);
                    return;
                }
            }
        }

        resolve();
    }

    return s;
}