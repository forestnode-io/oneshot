import { writeHeader } from './writeHeader';
import { writeBody } from './writeBody';
import { HTTPHeader } from '../types';
import { parseStatusLine, parseHeader } from '../util';
import { boundary, BufferedAmountLowThreshold } from './constants';

// rtcFetchFactory returns a function that can be used as an almost drop-in replacement for the fetch API
export function rtcFetchFactory(dc: RTCDataChannel, basicAuthToken?: string): (resource: RequestInfo | URL, options?: RequestInit | undefined) => Promise<Response> {
    dc.bufferedAmountLowThreshold = BufferedAmountLowThreshold;
    dc.binaryType = 'arraybuffer';
    let f = (resource: RequestInfo | URL, options?: RequestInit | undefined, progCallback?: (n: number, total?: number) => Promise<void>): Promise<Response> => {
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
        const chan = new MessageChannel();
        var buf: ArrayBuffer[] = [];

        dc.onmessage = (event: MessageEvent) => {
            // parse the response
            if (event.data instanceof ArrayBuffer) {
                // reading body which is sent as an ArrayBuffer
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

                    buf.push(event.data);
                    chan.port1.postMessage(null);

                    const h = new Headers();
                    for (const key in header) {
                        h.append(key, header[key]);
                    }
                    if (h.has('Content-Length') && progCallback) {
                        const contentLength = parseInt(h.get('Content-Length')!);
                        progCallback(-1, contentLength);
                    }

                    let responseInit: ResponseInit = {
                        status: status,
                        statusText: statusText,
                        headers: h,
                    };
                    let responseBody = new ReadableStream<Uint8Array>({
                        type: 'bytes',
                        start(controller) {},
                        pull(controller) {
                            return new Promise((resolve, reject) => {
                                chan.port2.onmessage = (event: MessageEvent) => {
                                    const s = buf.shift();
                                    if (!s) {
                                        // let the server know we got the eof
                                        dc.send("");
                                        controller.close();
                                        resolve();
                                        progCallback?.(0);
                                        return;
                                    }
                                    const n = s.byteLength;
                                    controller.enqueue(new Uint8Array(s));
                                    resolve();
                                    progCallback?.(n);
                                }
                            })
                        },
                    });
                    let response = new Response(responseBody, responseInit);
                    requestPromiseResolve(response);
                } else {
                    const blob = event.data as ArrayBuffer;
                    if (0 < blob.byteLength) {
                        buf.push(blob);

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

        if (!options) {
            options = {};
        }
        if (!options.headers) {
            options.headers = new Headers();
        }

        if (options?.body instanceof FormData) {
            if (options.headers instanceof Headers) {
                options.headers.append('Content-Type', 'multipart/form-data; boundary=' + boundary);
            } else if (typeof options.headers === 'object') {
                if (options.headers instanceof Array) {
                    options.headers.push(['Content-Type', 'multipart/form-data; boundary=' + boundary]);
                } else {
                    options.headers['Content-Type'] = 'multipart/form-data; boundary=' + boundary;
                }
            }
        }

        if (basicAuthToken) {
            if (options.headers instanceof Headers) {
                options.headers.append('X-HTTPOverWebRTC-Authorization', basicAuthToken);
            } else if (typeof options.headers === 'object') {
                if (options.headers instanceof Array) {
                    options.headers.push(['X-HTTPOverWebRTC-Authorization', basicAuthToken]);
                } else {
                    options.headers['X-HTTPOverWebRTC-Authorization'] = basicAuthToken;
                }
            }
        }

        writeHeader(dc, resource, options).then(() => {
            writeBody(dc, options?.body).catch((err: any) => {
                console.log("rejecting promise 1: ", err)
                requestPromiseReject(err);
            });
        }).catch((err: any) => {
            console.log("rejecting promise 2: ", err)
            requestPromiseReject(err);
        });

        return p;
    }

    return f;
}

