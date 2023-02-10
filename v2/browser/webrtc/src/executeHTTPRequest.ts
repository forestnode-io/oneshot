import { activateScriptTags } from './activateScriptTags';
import { triggerDownload } from './triggerDownload';
import { HTTPHeader } from './httpHeader';

const text_MIMERegexp = /^text\/.*$/;

export function executeHTTPRequest(channel: RTCDataChannel, request: string) {
    channel.send(request);
    channel.onmessage = (event: MessageEvent) => {
        // parse the response
        const splitPosition = event.data.search('\n\n');
        const status = event.data.slice(0, splitPosition);
        const body = event.data.slice(splitPosition + 2);

        console.log(`received http response with status section: ${status}`);

        var header: HTTPHeader = {};
        status.split('\n').forEach((line: string) => {
            const splitPosition = line.search(':');
            const key = line.slice(0, splitPosition);
            const value = line.slice(splitPosition + 2);
            header[key] = value;
        });
        var ct = header['Content-Type'];
        var cd = header['Content-Disposition'];

        // assume the content type is text if it is not specified
        if (!ct) {
            ct = 'text/plain';
        }

        // check if the content disposition is an attachment
        // if so, trigger a download
        const filename = filenameFromContentDisposition(cd);
        if (filename) {
            console.log(`triggering download of ${filename}`);
            triggerDownload(body, filename);
            return;
        }

        

        // otherwise, check if the content type is text
        // if so, display the body in the browser
        // otherwise, display the body as a preformatted text
        if (ct.match(text_MIMERegexp)) {
            if (ct === 'text/html') {
                document.body.innerHTML = body;
                activateScriptTags(document.body)
            } else {
                document.body.innerText = body;
                document.body.innerHTML = `<pre>${body}</pre>`;
            }
        } else {
            console.log(`falling back to displaying body as preformatted text`);
            document.body.innerText = body;
            document.body.innerHTML = `<pre>${body}</pre>`;
        }
    }
}

function filenameFromContentDisposition(cd: string): string {
    // check if the content disposition is an attachment
    if (!cd || !cd.includes('attachment')) {
        return "";
    }

    const filenameRegex = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/;
    const matches = filenameRegex.exec(cd);
    if (matches != null && matches[1]) {
        return matches[1].replace(/['"]/g, '');
    }
    return "";
}