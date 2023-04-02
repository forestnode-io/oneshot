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
//
// Basic Auth is handled via a Basic Auth token that is set in the offer passed to 
// the answerOffer method.
// The token is used by the fetch function in all subsequent HTTPOverWebRTC requests to the server.
// The server will accept this token in place of a username and password basic auth header.
// This prevents the discovery server from needing the credentials in order to pass them on the the clients.
// This also allows for the user to still interact with native browser dialogs by having the 
// auth dialog presented by the discovery server when this client code is being requested.
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

    constructor(rtcConfig: RTCConfiguration) {
        this.peerConnection = new RTCPeerConnection(rtcConfig);

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
            console.log("event", event)

            let target = event.target as RTCPeerConnection;
            console.log("target", target)
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
            console.log("oniceconnectionstatechange", event)
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

        // Extract the BasicAuthToken from the SDP offer
        let sdpParts = offer.sdp.match(/a=BasicAuthToken:(.*)/)
        if (sdpParts && sdpParts.length > 1) {
            this.baToken = sdpParts[1];
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