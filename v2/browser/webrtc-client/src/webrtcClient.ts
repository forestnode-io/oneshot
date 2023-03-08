import { rtcFetchFactory } from "./fetch/factory";

export class WebRTCClient {
    private peerConnection: RTCPeerConnection | undefined;
    private answered: boolean = false;

    private connectionPromiseResolve: () => void = () => { };
    private connectionPromise: Promise<void>;

    onAnswer: (answer: RTCSessionDescription) => void;

    constructor(iceServerURL: string, onAnswer: (answer: RTCSessionDescription) => void | undefined) {
        this.onAnswer = onAnswer;
        this.peerConnection = new RTCPeerConnection({
            iceServers: [{ urls: iceServerURL }],
        });

        this.connectionPromise = new Promise<void>((resolve, reject) => {
            this.connectionPromiseResolve = resolve;
        });

        this._configurePeerConnection();
    }

    private _configurePeerConnection(): void {
        const pc = this.peerConnection!;
        pc.onicegatheringstatechange = (event: Event) => {
            console.log("onicegatheringstatechange", pc.iceGatheringState);

            let target = event.target as RTCPeerConnection;
            if (target.iceGatheringState === 'complete' && target.localDescription) {
                this.onAnswer(target.localDescription);
            }
        };

        pc.ondatachannel = (event: RTCDataChannelEvent) => {
            console.log("ondatachannel", event);
            window.fetch = rtcFetchFactory(event.channel);
            window.rtcReady = true;
            this.connectionPromiseResolve();
        };

        pc.onnegotiationneeded = (event: Event) => {
            console.log("onnegotiationneeded");
        };

        pc.onsignalingstatechange = (event: Event) => {
            console.log("onsignalingstatechange", pc.signalingState);
        };

        pc.oniceconnectionstatechange = (event) => {
            console.log("oniceconnectionstatechange", pc.iceConnectionState);
        };

        pc.onicecandidate = (event: RTCPeerConnectionIceEvent) => {
            console.log("onicecandidate", event);
        }
    }

    public async answerOffer(offer: RTCSessionDescription): Promise<void> {
        if (this.answered) {
            return this.connectionPromise;
        }

        const pc = this.peerConnection!;
        try {
            await pc.setRemoteDescription(offer);
            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
            this.answered = true;
        } catch (err) {
            console.error(err);
        }

        return this.connectionPromise;
    }
}