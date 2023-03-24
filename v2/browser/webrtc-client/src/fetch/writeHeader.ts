import { DataChannelMTU } from "./constants";

export async function writeHeader(channel: RTCDataChannel, resource: RequestInfo | URL, options?: RequestInit): Promise<void> {
    const method = options?.method || 'GET';
    let headerString = `${method} ${resource} HTTP/1.1\n`;

    let headers = options?.headers ? options!.headers! : new Headers();
    if (headers instanceof Headers) {
        if (!headers.has('User-Agent')) {
            headers.append('User-Agent', navigator.userAgent);
        }
        console.log("writing HTTPOverWebRTC header: ", headers)
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
        } else {
            if (!headers['User-Agent']) {
                headers['User-Agent'] = navigator.userAgent;
            }
            for (const key in headers) {
                headerString += `${key}: ${headers[key]}\n`;
            }
        }
    }
    headerString += '\n';

    var pResolve: (() => void) | undefined = undefined;
    var pReject: ((reason: any) => void) | undefined = undefined;
    var p = new Promise<void>((resolve, reject) => {
        pResolve = resolve;
        pReject = reject;
    });

    const pump = sendPump(channel, pResolve!, headerString);
    pump();

    return p;
}

function sendPump(channel: RTCDataChannel, resolve: (() => void), data: string): () => void {
    var mtu = DataChannelMTU;
    const s = function () {
        while (data.length) {
            if (channel.bufferedAmount > channel.bufferedAmountLowThreshold) {
                channel.onbufferedamountlow = () => {
                    channel.onbufferedamountlow = null;
                    s();
                }
            }

            if (data.length < mtu) {
                mtu = data.length;
            }

            const chunk = data.slice(0, mtu);
            data = data.slice(mtu);
            channel.send(chunk);

            if (mtu != DataChannelMTU) {
                resolve();
                return;
            }
        }
    }

    return s;
}