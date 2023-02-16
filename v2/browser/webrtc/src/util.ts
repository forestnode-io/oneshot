import { HTTPHeader, StatusLine } from './types';

export function parseHeader(header: string): HTTPHeader {
    const lines = header.split('\n');
    const h: HTTPHeader = {};
    for (const line of lines) {
        const splitPosition = line.search(':');
        if (splitPosition === -1) {
            continue;
        }
        const key = line.slice(0, splitPosition);
        const value = line.slice(splitPosition + 1);
        h[key] = value.trim();
    }
    return h;
}

export function parseStatusLine(line: string): StatusLine {
    if (!line.startsWith('HTTP/1.1')) {
        throw new Error(`unexpected status line: ${line}`);
    }
    const statusLineSplit = line.split(' ');
    if (statusLineSplit.length < 3) {
        throw new Error(`unexpected status line: ${line}`);
    }
    const status = parseInt(statusLineSplit[1]);
    const statusText = statusLineSplit.slice(2).join(' ');
    return { status, statusText };
}

export async function formDataToString(formData: FormData, boundary: string): Promise<string> {
    let p = new Promise<string>(async (resolve, reject) => {
        let result = "";

        for (const pair of formData.entries()) {
            const name = pair[0];
            const stringOrFile = pair[1];

            result += `--${boundary}\n`;
            if (typeof stringOrFile === 'string') {
                console.log(`string name: ${name}`);
                result += `Content-Disposition: form-data; name="${name}"\n\n`;
                result += `${stringOrFile as string}\n`;
            } else {
                const file = stringOrFile as File;
                result += `Content-Disposition: form-data; name="${name}"; filename="${file.name}"\n`;
                if (file.type) {
                    result += `Content-Type: ${file.type}\n`;
                }

                console.log(`file name: ${file.name}`);
                const decoder = new TextDecoder();
                const payload = decoder.decode(await file.arrayBuffer());
                result += `\n${payload}\n`;
            }
        }

        if (result) {
            result += `--${boundary}--\n`;
        }

        resolve(result);
    });

    return p;
}