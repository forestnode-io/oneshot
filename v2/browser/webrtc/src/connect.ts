import { HTTPHeader } from "./httpHeader";
import { WebRTCClient } from "./webrtcClient";
import { WebRTCXMLHttpRequest } from "./webRTCXMLHttpRequest";

function connect() {
    const iceURL = (document.getElementById('ice-server-url') as HTMLInputElement)?.value;
    const rsdString = (document.getElementById('server-sd') as HTMLInputElement)?.value;
    const rsd = rsdString ? JSON.parse(rsdString) : undefined;

    if (!rsd || !iceURL) {
        alert('missing server-sd or ice-url');
        return;
    }

    const c = new WebRTCClient(iceURL, rsd);
    c.onAnswer = (answer: RTCSessionDescription) => {
        const answerString = JSON.stringify(answer);
        const answerEl = document.getElementById('answer-container');

        console.log(`answer session description:\n${answerString}`);

        if (answerEl) {
            answerEl.innerText = answerString;
        }

        navigator.clipboard.writeText(answerString);
    };
    const req = (document.getElementById('httpRequest') as HTMLInputElement)?.value;
    if (!req) {
        alert('missing request');
        return;
    }

    c.exec(req);
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
        WebRTCXMLHttpRequest: Function; 
        oneshotWebRTCDataChannel: RTCDataChannel | undefined;
    }
}

window.WebRTCXMLHttpRequest = WebRTCXMLHttpRequest;