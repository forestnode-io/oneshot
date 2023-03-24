import { rtcFetchFactory } from "./fetch/factory";
import { visit } from "./browser/visit";

type AnswerOfferResponse = {
    Answer: Promise<RTCSessionDescription>;
    ConnectionEstablished: Promise<void>;
}

// HTTPOverWebRTCClient establishes a connection with a remote server over WebRTC.
// Once the connection is established, the global fetch function is replaced with
// a function that uses the WebRTC data channel to send HTTP requests to the remote
// server.
export class HTTPOverWebRTCClient {
    private peerConnection: RTCPeerConnection | undefined;
    private answered: boolean = false;

    private connectionPromiseResolve: () => void = () => { };
    private connectionPromiseReject: (reason: any) => void = () => { };
    private connectionPromise: Promise<void>;

    private resolveAnswerPromise: (answer: RTCSessionDescription) => void = () => { };
    private rejectAnswerPromise: (reason: any) => void = () => { };
    private answerPromise: Promise<RTCSessionDescription> = new Promise<RTCSessionDescription>((resolve, reject) => {
        this.resolveAnswerPromise = resolve;
        this.rejectAnswerPromise = reject;
    });

    private _fetch: ((resource: RequestInfo | URL, options?: RequestInit | undefined) => Promise<Response>) | undefined;

    private baToken: string | undefined;

    constructor(rtcConfig: RTCConfiguration, basicAuthToken?: string) {
        //this.onAnswer = onAnswer;
        this.peerConnection = new RTCPeerConnection(rtcConfig);
        if (basicAuthToken) {
            this.baToken = basicAuthToken;
        }

        this.connectionPromise = new Promise<void>((resolve, reject) => {
            this.connectionPromiseResolve = resolve;
            this.connectionPromiseReject = reject;
        });

        this._configurePeerConnection();
    }

    private _configurePeerConnection(): void {
        const pc = this.peerConnection!;
        pc.onicegatheringstatechange = (event: Event) => {
            console.log("onicegatheringstatechange", pc.iceGatheringState);

            let target = event.target as RTCPeerConnection;
            if (target.iceGatheringState === 'complete' && target.localDescription) {
                this.resolveAnswerPromise(target.localDescription);
                //this.onAnswer(target.localDescription);
            }
        };

        pc.ondatachannel = (event: RTCDataChannelEvent) => {
            console.log("ondatachannel", event);
            this._fetch = rtcFetchFactory(event.channel, this.baToken);
            window.fetch = this._fetch!;

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

    // answerOffer returns a pair of promises.
    //
    // The first promise resolves to the answer to the offer.
    // The answer needs to be sent to the remote peer somehow,
    // this is known as the "signaling" step and is not handled by this class.
    //
    // The second promise resolves when the connection is established and the
    // global fetch function is replaced with a function that uses the WebRTC
    // data channel to send HTTP requests to the remote server.
    public answerOffer(offer: RTCSessionDescription): AnswerOfferResponse {
        if (this.answered) {
            return {
                Answer: Promise.reject("already answered"),
                ConnectionEstablished: Promise.reject("already answered")
            };
        }

        const pc = this.peerConnection!;
        const consumeAnswer = (answer: RTCSessionDescriptionInit) => {
            pc.setLocalDescription(answer).then(() => {
                this.answered = true;
                this.resolveAnswerPromise(new RTCSessionDescription(answer));
            });
        };

        pc.setRemoteDescription(offer).
            then(() => pc.createAnswer().then(consumeAnswer)).
            catch((err) => {
                console.error(err);
                this.rejectAnswerPromise(err);
                this.connectionPromiseReject(err);
            });

        return {
            Answer: this.answerPromise,
            ConnectionEstablished: this.connectionPromise
        };
    }

    // fetch simulates the global fetch function but uses the WebRTC data channel.
    // This function is only available after the connection is established.
    public fetch(resource: RequestInfo | URL, options?: RequestInit | undefined): Promise<Response> {
        if (!this._fetch) {
            return Promise.reject("HTTPOverWebRTC fetch not ready");
        }

        return this._fetch(resource, options);
    }

    public visit(request: RequestInfo | URL, options?: RequestInit | undefined): Promise<void> {
        if (!this._fetch) {
            return Promise.reject("HTTPOverWebRTC fetch not ready");
        }

        return visit(request, options, this._fetch!.bind(this));
    }

    public connected(): boolean {
        return this._fetch !== undefined;
    }
}