import { DataChannelMTU } from "./constants";

export async function writeHeader(channel: RTCDataChannel, resource: RequestInfo | URL, options?: RequestInit): Promise<void> {
    console.log('writeHeader: ', resource, options)
    const method = options?.method || 'GET';
    let headerString = `${method} ${resource} HTTP/1.1\n`;

    if (options?.headers) {
        if (options.headers instanceof Headers) {
            options.headers.forEach((value, key) => {
                headerString += `${key}: ${value}\n`;
            });
        } else if (typeof options.headers === 'object') {
            if (options.headers instanceof Array) {
                for (var i = 0 ; i < options.headers.length; i++) {
                    headerString += `${options.headers[i][0]}: ${options.headers[i][1]}\n`;
                }
            } else {
                for (const key in options.headers) {
                    headerString += `${key}: ${options.headers[key]}\n`;
                }
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