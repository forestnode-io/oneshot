import { executeHTTPRequest } from "./executeHTTPRequest";

export class WebRTCClient {
    iceServerURL: string;
    remoteSessionDescription: RTCSessionDescription;
    peerConnection: RTCPeerConnection | undefined;
    onAnswer: ((answer: RTCSessionDescription) => void) = (answer: RTCSessionDescription) => { 
        console.log(`answer session description:\n${answer}`);
    };

    constructor(iceServerURL: string, remoteSessionDescription: RTCSessionDescription) {
        this.iceServerURL = iceServerURL;
        this.remoteSessionDescription = remoteSessionDescription;
    }

    async exec(request: string) {
        this.peerConnection = new RTCPeerConnection({
            iceServers: [{ urls: this.iceServerURL }],
        });

        const pc = this.peerConnection;

        pc.onicegatheringstatechange = (event: Event) => {
            console.log("onicegatheringstatechange", pc.iceGatheringState);

            let target = event.target as RTCPeerConnection;
            if (target.iceGatheringState === 'complete' && target.localDescription) {
                if (this.onAnswer) {
                    this.onAnswer(target.localDescription);
                }
            }
        };

        pc.ondatachannel = (event: RTCDataChannelEvent) => {
            console.log("ondatachannel", event);

            const channel = event.channel;
            channel.onopen = (event: Event) => {
                console.log("onopen", event);
                executeHTTPRequest(channel, request);
            };
            channel.onclose = (event: Event) => {
                console.log("onclose", event);
            }
        };

        setPeerConnectionStubs(pc);

        try {
            await pc.setRemoteDescription(this.remoteSessionDescription);
            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
        } catch (err) {
            console.error(err);
        }

        return;
    }
}

function setPeerConnectionStubs(pc: RTCPeerConnection) {
    pc.onnegotiationneeded = (event: Event) => {
        console.log("onnegotiationneeded");
    };

    pc.onsignalingstatechange = (event: Event) => {
        console.log("onsignalingstatechange", pc.signalingState);
    };

    pc.oniceconnectionstatechange = (event) => {
        console.log("oniceconnectionstatechange", pc.iceConnectionState);
    };

}