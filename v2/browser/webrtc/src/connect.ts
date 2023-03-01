import { visit } from "./browser/visit";
import { WebRTCClient } from "./webrtcClient";

export function autoOnAnswerFactory(sessionID: number | undefined): (answer: RTCSessionDescription) => void {
    return (answer: RTCSessionDescription) => {
        fetch('/api/sdp/answer', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                "id": sessionID,
                "answer": answer.sdp,
            })
        }).then((res) => {
            if (res.status !== 200) {
                alert('failed to send answer: ' + res.status + ' ' + res.statusText);
            }
        }).catch((err) => {
            alert(`failed to send answer: ${err}`);
        });
    };
};

export function manualOnAnswer(answer: RTCSessionDescription): void {
    const answerString = JSON.stringify(answer);
    const answerEl = document.getElementById('answer-container');

    if (answerEl) {
        answerEl.innerText = answerString;
    }

    navigator.clipboard.writeText(answerString);
}

export type connectConfig = {
    onAnswer: (answer: RTCSessionDescription) => void;
    iceURL: string;
    offer: string;
    sessionID: number | undefined;
}

export function connect(config: connectConfig) {
    const offerJSON = config.offer ? JSON.parse(config.offer) : undefined;
    if (!offerJSON || !config.iceURL) {
        alert('missing offer or ice-url');
        return;
    }
    new WebRTCClient(config.iceURL, config.onAnswer).
        answerOffer(offerJSON as RTCSessionDescription).then(() => {
            visit('/', {})
        });
}