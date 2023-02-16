import { sendHeader } from './sendHeader';
import { sendBody } from './sendBody';
import { HTTPHeader } from '../types';
import { parseStatusLine, parseHeader } from '../util';

// rtcFetchFactory returns a function that can be used as an almost drop-in replacement for the fetch API
export function rtcFetchFactory(dc: RTCDataChannel): (resource: RequestInfo | URL, options?: RequestInit | undefined) => Promise<Response> {
    return (resource: RequestInfo | URL, options?: RequestInit | undefined): Promise<Response> => {
        var requestPromiseResolve: (value: Response) => void = () => { };
        var requestPromiseReject: (reason?: any) => void = () => { };
        const p = new Promise<Response>((resolve, reject) => {
            requestPromiseResolve = resolve;
            requestPromiseReject = reject;
        });

        var headerBuf = '';
        var status: number = -1;
        var statusText: string = '';
        var header: HTTPHeader = {};
        let chan = new MessageChannel();
        var buf: ArrayBuffer[] = [];

        dc.onmessage = async (event: MessageEvent) => {
            // parse the response
            if (event.data instanceof Blob) {
                // reading body which is sent as a blob
                // check if header has been parsed, if not, parse it
                // shave off the status line
                if (status === -1) {
                    const statusLine = headerBuf.slice(0, headerBuf.search('\n'));
                    headerBuf = headerBuf.slice(headerBuf.search('\n') + 1);
                    // parse the status line
                    const sl = parseStatusLine(statusLine);
                    status = sl.status;
                    statusText = sl.statusText;

                    // parse the cached header string
                    header = parseHeader(headerBuf);
                    headerBuf = '';

                    let ab = await (event.data as Blob).arrayBuffer();
                    buf.push(ab);
                    chan.port1.postMessage(null);

                    var h = new Headers();
                    for (const key in header) {
                        h.append(key, header[key]);
                    }

                    let responseInit: ResponseInit = {
                        status: status,
                        statusText: statusText,
                        headers: h,
                    };
                    let responseBody = new ReadableStream<Uint8Array>({
                        type: 'bytes',
                        start(controller) {
                            if (controller instanceof ReadableByteStreamController) {
                                if (controller.byobRequest) {
                                    throw new Error('byobRequest not supported');
                                }
                            }
                        },
                        pull(controller) {
                            return new Promise((resolve, reject) => {
                                chan.port2.onmessage = (event: MessageEvent) => {
                                    const s = buf.shift();
                                    if (!s) {
                                        controller.close();
                                        resolve();
                                        return;
                                    }
                                    controller.enqueue(new Uint8Array(s));
                                    resolve();
                                }
                            })
                        },
                    });
                    let response = new Response(responseBody, responseInit);
                    requestPromiseResolve(response);
                } else {
                    const blob = event.data as Blob;
                    if (blob.size !== 0) {
                        let ab = await blob.arrayBuffer();
                        buf.push(ab);
                        chan.port1.postMessage(null);
                    }
                }
            } else if (typeof event.data === 'string') {
                if (status === -1) {
                    headerBuf += event.data;
                } else {
                    // oneshot sends a string after the header to indicate the end of the response
                    chan.port1.postMessage(null);
                }
            }
        }

        sendHeader(dc, resource, options);
        sendBody(dc, options?.body);
        return p;
    }
}

