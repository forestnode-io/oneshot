import { visit } from "./browser/visit";
import { HTTPHeader } from "./types";
import { WebRTCClient } from "./webrtcClient";

function connect() {
    const iceURL = (document.getElementById('ice-server-url') as HTMLInputElement)?.value;
    const rsdString = (document.getElementById('server-sd') as HTMLInputElement)?.value;
    const rsd = rsdString ? JSON.parse(rsdString) : undefined;

    if (!rsd || !iceURL) {
        alert('missing server-sd or ice-url');
        return;
    }

    const c = new WebRTCClient(iceURL, (answer: RTCSessionDescription) => {
        const answerString = JSON.stringify(answer);
        const answerEl = document.getElementById('answer-container');

        console.log(`answer session description:\n${answerString}`);

        if (answerEl) {
            answerEl.innerText = answerString;
        }

        navigator.clipboard.writeText(answerString);
    });

    const req = (document.getElementById('httpRequest') as HTMLInputElement)?.value;
    if (!req) {
        alert('missing request');
        return;
    }

    c.answerOffer(rsd as RTCSessionDescription).then(() => {
        visit('/', {})
    });
}

document.getElementById('connect-button')?.addEventListener('click', connect);
const hrEl = document.getElementById('httpRequest') as HTMLInputElement;
if (hrEl) {
    const headers: HTTPHeader = {};
    headers['User-Agent'] = navigator.userAgent;
    let header = "";

    for (const key in headers) {
        header += `${key}: ${headers[key]}\n`;
    }

    if (header) {
        header += "\n";
    }

    hrEl.value = `GET / HTTP/1.1\n${header}`;
}

declare global {
    interface Window { 
        WebRTCClient: Function;
        rtcReady: boolean;
    }
}

window.WebRTCClient = WebRTCClient;