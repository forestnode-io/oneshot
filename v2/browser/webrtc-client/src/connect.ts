import { HTTPOverWebRTCClient } from "./webrtcClient";

export function autoOnAnswerFactory(endpoint: string, sessionID: string | undefined): (answer: RTCSessionDescription) => void {
    return (answer: RTCSessionDescription) => {
        fetch(endpoint, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                "SessionID": sessionID,
                "Answer": answer.sdp,
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
    rtcConfig: RTCConfiguration;
    offer: RTCSessionDescription;
    sessionID: string | undefined;
    endpoint: string;
    baToken: string | undefined;
}

export function connect(config: connectConfig) {
    if (!config.offer) {
        alert('no offer');
        return;
    }

    const client = new HTTPOverWebRTCClient(config.rtcConfig, config.baToken);
    const resp = client.answerOffer(config.offer);
    resp.ConnectionEstablished.then(() => client.visit('/', {}));
    resp.Answer.then(config.onAnswer);
}