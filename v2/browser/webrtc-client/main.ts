import { WebRTCClient } from './src/webrtcClient';
import { autoOnAnswerFactory, manualOnAnswer, connect, connectConfig } from './src/connect';

var button = document.getElementById('connect-button');

declare global {
    interface Window {
        WebRTCClient: Function;
        rtcReady: boolean;
    }
}

window.WebRTCClient = WebRTCClient;

function main() {
    if (button) {
        button.onclick = () => {
            connect(getConfig(false));
        };
    } else {
        const span = document.createElement('span');
        span.innerText = 'tunnelling to oneshot server...';
        document.body.appendChild(span);

        connect(getConfig(true));
    }
}

function getConfig(autoAnswer: boolean): connectConfig {
    const config: connectConfig = {} as connectConfig;

    var el = (document.getElementById('ice-server-url') as HTMLInputElement);
    config.iceURL = el.value;
    el.parentNode?.removeChild(el);

    var el = (document.getElementById('session-id') as HTMLInputElement);
    config.sessionID = parseInt(el.value);
    el.parentNode?.removeChild(el);

    var el = (document.getElementById('offer-sdp') as HTMLInputElement);
    config.offer = el.value;
    el.parentNode?.removeChild(el);
    
    if (autoAnswer) {
        config.onAnswer = autoOnAnswerFactory(config.sessionID);
    }  else {
        config.onAnswer = manualOnAnswer;
    }

    return config;
}

main();