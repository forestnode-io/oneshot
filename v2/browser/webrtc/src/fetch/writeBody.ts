import { boundary, DataChannelMTU } from './constants';

export async function writeBody(channel: RTCDataChannel, body: BodyInit | null | undefined): Promise<void> {
    if (!body) {
        return Promise.resolve();
    }

    var buf: ArrayBuffer | undefined = undefined;
    var pResolve: (() => void) | undefined = undefined;
    var pReject: ((reason: any) => void) | undefined = undefined;
    var p = new Promise<void>((resolve, reject) => {
        pResolve = resolve;
        pReject = reject;
    });

    if (body instanceof ReadableStream) {
        const streamContents = await body.getReader().read();
        body = streamContents.value;
    } else {
        // body is XMLHttpRequestBodyInit
        if (body instanceof FormData) {
            pumpForm(channel, body).then(() => {
                pResolve!();
            });
            return p;
        } else if (body instanceof Blob) {
            buf = await body.arrayBuffer();
        } else if (body instanceof URLSearchParams) {
            buf = new TextEncoder().encode(body.toString());
        } else if (typeof body === 'string') {
            buf = new TextEncoder().encode(body);
        } else if (body instanceof ArrayBuffer) {
            buf = body;
        }
    }

    const pump = sendPump(channel, buf!, pResolve);
    pump();

    return p;
}

function sendPump(channel: RTCDataChannel, data: ArrayBuffer, resolve?: (() => void)): () => void {
    var mtu = DataChannelMTU;
    const s = function () {
        while (data.byteLength) {
            if (channel.bufferedAmount > channel.bufferedAmountLowThreshold) {
                channel.onbufferedamountlow = () => {
                    channel.onbufferedamountlow = null;
                    s();
                }
            }

            if (data.byteLength < mtu) {
                mtu = data.byteLength;
            }

            const chunk = data.slice(0, mtu);
            data = data.slice(mtu);
            channel.send(chunk);

            if (mtu != DataChannelMTU) {
                if (resolve) resolve();
                return;
            }
        }
    }

    return s;
}


async function pumpForm(channel: RTCDataChannel, form: FormData): Promise<void> {
    const encoder = new TextEncoder();

    return new Promise<void>(async (resolve, reject) => {
        for (const pair of form.entries()) {
            var buf = `--${boundary}\n`;
            const name = pair[0];
            const stringOrFile = pair[1];
            if (typeof stringOrFile === 'string') {
                buf += `Content-Disposition: form-data; name="${name}"\n\n`;
                channel.send(encoder.encode(buf));
                channel.send(encoder.encode(stringOrFile));
            } else {
                const file = stringOrFile as File;
                buf += `Content-Disposition: form-data; name="${name}"; filename="${file.name}"\n`;
                if (file.type) {
                    buf += `Content-Type: ${file.type}\n\n`;
                } else {
                    buf += 'Content-Type: application/octet-stream\n\n';
                }
                channel.send(encoder.encode(buf));

                var fileResolve: (() => void) | undefined;
                var filePromise = new Promise<void>((resolve, reject) => {
                    fileResolve = resolve;
                });
                const fileReader = new FileReader();
                var offset = 0;
                fileReader.onerror = (e) => {
                    console.log('Error reading file', e);
                }
                fileReader.onabort = (e) => {
                    console.log('File reading aborted', e);
                }
                fileReader.onload = (e) => {
                    const r = e.target!.result as ArrayBuffer;
                    channel.send(r);
                    offset += r.byteLength;
                    if (offset < file.size) {
                        loadNextFileChunk(offset);
                    } else {
                        fileResolve!();
                    }
                };
                const loadNextFileChunk = (o: number) => {
                    fileReader.readAsArrayBuffer(file.slice(o, o + DataChannelMTU));
                }
                loadNextFileChunk(offset);
                await filePromise;
            }
        }

        channel.send(encoder.encode(`\n--${boundary}--\n`));
        channel.send("EOF")

        resolve();
    });
}