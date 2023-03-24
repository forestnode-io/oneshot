import { activateScriptTags } from './activateScriptTags';
import { triggerDownload } from './triggerDownload';

const text_MIMERegexp = /^text\/.*$/;

export async function visit(request: RequestInfo | URL,
    options?: RequestInit | undefined,
    fetcher: ((request: RequestInfo | URL, options?: RequestInit | undefined) => Promise<Response>) = fetch,
    ): Promise<void> {
    const spinnerEl = document.createElement('span');
    document.body.innerHTML = '';
    document.body.appendChild(spinnerEl);

    var contentLength = 0;
    var downloaded = 0;
    const progCallback = (n: number, total?: number): Promise<void> => {
        if (n === -1 && total) {
            contentLength = total;
        } else if (0 < n) {
            downloaded += n;
            if (contentLength) {
                const percent = (downloaded / contentLength * 100).toFixed(2);
                spinnerEl.innerText = `receiving data: ${percent}%`;
            } else {
                spinnerEl.innerText = `receiving data: ${downloaded} bytes`;
            }
        } else {
            spinnerEl.innerText = `receiving data: done`;
        }

        return Promise.resolve();
    };
    interface progFetchIface {
        (request: RequestInfo | URL, options?: RequestInit | undefined, progCallback?: (n: number, total?: number) => Promise<void>): Promise<Response>;
    }
    const progFetch = fetcher as progFetchIface;
    var resp = await progFetch(request, options, progCallback);
    const header = resp.headers!;
    var ct = header.get('Content-Type') ? header.get('Content-Type')! : '';
    ct = ct.split(';')[0];
    var cd = header.get('Content-Disposition') ? header.get('Content-Disposition')! : '';
    if (!ct) {
        ct = 'text/plain';
    }

    // check if the content disposition is an attachment
    // if so, trigger a download
    const filename = filenameFromContentDisposition(cd);
    if (filename) {
        const bodyBlob = await resp.blob();
        triggerDownload(bodyBlob, filename);
        return;
    }

    // otherwise, check if the content type is text
    // if so, display the body in the browser
    // otherwise, display the body as a preformatted text
    if (ct.match(text_MIMERegexp)) {
        const body = await resp.text();
        if (ct === 'text/html') {
            const parser = new DOMParser();
            const dom = parser.parseFromString(body, 'text/html');
            document.body = dom.body;
            activateScriptTags(document.body)
        } else {
            document.body.innerText = body;
            document.body.innerHTML = `<pre>${body}</pre>`;
        }
    } else if (ct.startsWith('application/')) {
        const body = await resp.blob();
        let file = new Blob([body], { type: ct });
        let fileURL = URL.createObjectURL(file);
        window.open(fileURL, "_self");
    } else {
        console.log(`falling back to displaying body as preformatted text`);
        const body = await resp.text();
        document.body.innerText = body;
        document.body.innerHTML = `<pre>${body}</pre>`;
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