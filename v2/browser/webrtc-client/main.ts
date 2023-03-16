import { WebRTCClient } from './src/webrtcClient';
import { autoOnAnswerFactory, manualOnAnswer, connect, connectConfig } from './src/connect';

declare global {
    interface Window {
        WebRTCClient: Function;
        rtcReady: boolean;
        config: {
            RTCConfigurationJSON: string;
            SessionID: string;
            OfferJSON: string;
            Endpoint: string;
        };
    }
}

window.WebRTCClient = WebRTCClient;

function main() {
    const button = document.getElementById('connect-button');

    if (button) {
        button.onclick = () => {
            try {
                connect(getConfig(false));
            } catch (e) {
                alert(e);
                return;
            }
        };
    } else {
        const span = document.createElement('span');
        span.innerText = 'tunnelling to oneshot server...';
        document.body.appendChild(span);

        try {
            connect(getConfig(true));
        } catch (e) {
            alert(e);
            return;
        }
    }
}

function getConfig(autoAnswer: boolean): connectConfig {
    const cconfig: connectConfig = {} as connectConfig;
    const config = window.config;
    if (!config) {
        throw new Error('no config');
    }

    cconfig.rtcConfig = JSON.parse(config.RTCConfigurationJSON);
    if (!cconfig.rtcConfig) {
        throw new Error('no rtc config');
    }
    if (!cconfig.rtcConfig.iceServers) {
        throw new Error('no ice servers');
    }
    cconfig.sessionID = config.SessionID;
    if (!cconfig.sessionID) {
        throw new Error('no session id');
    }
    cconfig.offer = JSON.parse(config.OfferJSON);
    if (!cconfig.offer) {
        throw new Error('no offer');
    }
    cconfig.endpoint = config.Endpoint;
    if (!cconfig.endpoint) {
        throw new Error('no endpoint');
    }

    if (autoAnswer) {
        cconfig.onAnswer = autoOnAnswerFactory(cconfig.endpoint, cconfig.sessionID);
    } else {
        cconfig.onAnswer = manualOnAnswer;
    }

    return cconfig;
}

main();