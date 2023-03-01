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