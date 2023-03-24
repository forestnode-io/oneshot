import { HTTPOverWebRTCClient } from './src/webrtcClient';
import { autoOnAnswerFactory, manualOnAnswer, connect, connectConfig } from './src/connect';

declare global {
    interface Window {
        WebRTCClient: Function;
        rtcReady: boolean;
        sessionToken: string;
        basicAuthToken: string | undefined;
    }
}

// export webrtc client to global scope
window.WebRTCClient = HTTPOverWebRTCClient;

type Config = {
    RTCConfiguration: RTCConfiguration;
    RTCSessionDescription: RTCSessionDescription;
    SessionID: string;
};

function fetchConfig(): Promise<Config> {
    let opts: RequestInit = {
        credentials: 'same-origin',
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
            'X-Session-Token': window.sessionToken,
        },
    }
    return fetch(window.location.href, opts)
        .then((res) => {
            if (res.status !== 200) {
                throw new Error('failed to fetch config: ' + res.status + ' ' + res.statusText);
            }
            return res.json();
        })
        .then((json) => {
            if (!json.RTCSessionDescription || !json.RTCConfiguration || !json.SessionID) {
                throw new Error('failed to fetch config: invalid json');
            }

            return json as Config;
        })
        .catch((err) => {
            throw new Error(`failed to fetch config: ${err}`);
        });
}


async function main() {
    const button = document.getElementById('connect-button');

    if (button) {
        // no autoconnect, assign connection handler to button
        try {
            let conf = await getConfig(false);
            button.onclick = () => {
                try {
                    connect(conf);
                } catch (e) {
                    alert(e);
                    return;
                }
            };
        } catch (e) {
            alert(e);
            return;
        }
    } else {
        // autoconnect
        const span = document.createElement('span');
        span.innerText = 'tunnelling to oneshot server...';
        document.body.appendChild(span);

        try {
            let conf = await getConfig(true);
            console.log('got connection config: ', conf);
            connect(conf);
        } catch (e) {
            alert(e);
            return;
        }
    }
}

async function getConfig(autoAnswer: boolean): Promise<connectConfig> {
    let cconfig = {} as connectConfig;
    try {
        let remoteConfig = await fetchConfig();
        cconfig.rtcConfig = remoteConfig.RTCConfiguration;
        cconfig.offer = remoteConfig.RTCSessionDescription;
        cconfig.sessionID = remoteConfig.SessionID;
        cconfig.endpoint = window.location.href;
        cconfig.baToken = window.basicAuthToken;

        if (autoAnswer) {
            cconfig.onAnswer = autoOnAnswerFactory(cconfig.endpoint, cconfig.sessionID);
        } else {
            cconfig.onAnswer = manualOnAnswer;
        }
    } catch (e) {
        alert(e);
        return Promise.reject(e);
    }

    return cconfig;
}

main();