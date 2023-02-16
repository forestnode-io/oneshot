import { activateScriptTags } from './activateScriptTags';
import { triggerDownload } from './triggerDownload';
import { HTTPHeader } from '../types';

const text_MIMERegexp = /^text\/.*$/;

export async function visit(request: RequestInfo | URL, options?: RequestInit | undefined): Promise<void> {
    var resp = await fetch(request, options);
    const header = resp.headers!;
    var ct = header.get('Content-Type') ? header.get('Content-Type')! : '';
    var cd = header.get('Content-Disposition') ? header.get('Content-Disposition')! : '';

    if (!ct) {
        ct = 'text/plain';
    }

    // check if the content disposition is an attachment
    // if so, trigger a download
    const filename = filenameFromContentDisposition(cd);
    if (filename) {
        console.log(`triggering download of ${filename}`);
        triggerDownload(resp.body, filename);
        return;
    }

    // otherwise, check if the content type is text
    // if so, display the body in the browser
    // otherwise, display the body as a preformatted text
    const body  = await resp.text();
    if (ct.match(text_MIMERegexp)) {
        if (ct === 'text/html') {
            const parser = new DOMParser();
            const dom = parser.parseFromString(body, 'text/html');
            document.body = dom.body;
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