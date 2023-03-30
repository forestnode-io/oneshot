import { HTTPOverWebRTCClient } from './src/webrtcClient';

type Config = {
    RTCConfiguration: RTCConfiguration;
    RTCSessionDescription: RTCSessionDescription;
    SessionID: string;
};

declare global {
    interface Window {
        WebRTCClient: Function;
        rtcReady: boolean;
        sessionToken: string;
        basicAuthToken: string | undefined;
        config: Config | undefined;
    }
};

// export webrtc client to global scope
window.WebRTCClient = HTTPOverWebRTCClient;

