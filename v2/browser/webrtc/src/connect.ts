import { visit } from "./browser/visit";
import { HTTPHeader } from "./types";
import { WebRTCClient } from "./webrtcClient";

const offer = "{{ .Offer }}";
const sessionID = "{{ .SessionID }}";
const sessionIDNumber = parseInt(sessionID);

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

function signallingServerConnect() {
    const iceURL = (document.getElementById('ice-server-url') as HTMLInputElement)?.value;
    const rsdString = (document.getElementById('server-sd') as HTMLInputElement)?.value;
    const rsd = rsdString ? JSON.parse(rsdString) : undefined;

    if (!rsd || !iceURL) {
        alert('missing server-sd or ice-url');
        return;
    }

    const c = new WebRTCClient(iceURL, (answer: RTCSessionDescription) => {
        fetch('/api/sdp/answer', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                "id": sessionIDNumber,
                "answer": answer.sdp,
            })
        }).then((res) => {
            if (res.status !== 200) {
                alert('failed to send answer: ' + res.status + ' ' + res.statusText);
            }
        }).catch((err) => {
            alert(`failed to send answer: ${err}`);
        });
    });

    const req = (document.getElementById('httpRequest') as HTMLInputElement)?.value;
    if (!req) {
        alert('missing request');
        return;
    }

    c.answerOffer(rsd as RTCSessionDescription).then(() => {
        console.log('answer offer');
        visit('/', {})
    });
}

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

if (12 < offer.length) {
    console.log('offer is not default');
    const offerEl = document.getElementById('server-sd');
    if (offerEl) {
        console.log('setting offer');
        offerEl.innerText = offer;
    }

    signallingServerConnect();
} else {
    document.getElementById('connect-button')?.addEventListener('click', connect);
}


declare global {
    interface Window {
        WebRTCClient: Function;
        rtcReady: boolean;
    }
}

window.WebRTCClient = WebRTCClient;