"use strict";(()=>{var p=16384;var n=1*p;var e=8*p;var y="boundary";async function r(e,n,t){console.log("writeHeader: ",n,t);const o=(t==null?void 0:t.method)||"GET";let r=`${o} ${n} HTTP/1.1
`;let s=(t==null?void 0:t.headers)?t.headers:new Headers;if(s instanceof Headers){if(!s.has("User-Agent")){s.append("User-Agent",navigator.userAgent)}s.forEach((e,n)=>{r+=`${n}: ${e}
`})}else if(typeof s==="object"){if(s instanceof Array){var i=false;for(var a=0;a<s.length;a++){r+=`${s[a][0]}: ${s[a][1]}
`;if(s[a][0]==="User-Agent"){i=true}}if(!i){r+=`User-Agent: ${navigator.userAgent}
`}}else{if(!s["User-Agent"]){s["User-Agent"]=navigator.userAgent}for(const u in s){r+=`${u}: ${s[u]}
`}}}r+="\n";var c=void 0;var l=void 0;var f=new Promise((e,n)=>{c=e;l=n});const d=h(e,c,r);d();return f}function h(n,t,o){var r=p;const s=function(){while(o.length){if(n.bufferedAmount>n.bufferedAmountLowThreshold){n.onbufferedamountlow=()=>{n.onbufferedamountlow=null;s()}}if(o.length<r){r=o.length}const e=o.slice(0,r);o=o.slice(r);n.send(e);if(r!=p){t();return}}};return s}async function s(e,n){if(!n){e.send(new Uint8Array(0));e.send("");return Promise.resolve()}var t=void 0;var o=void 0;var r=void 0;var s=new Promise((e,n)=>{o=e;r=n});if(n instanceof ReadableStream){const a=await n.getReader().read();n=a.value}else{if(n instanceof FormData){l(e,n).then(()=>{o()});return s}else if(n instanceof Blob){t=await n.arrayBuffer()}else if(n instanceof URLSearchParams){t=(new TextEncoder).encode(n.toString())}else if(typeof n==="string"){t=(new TextEncoder).encode(n)}else if(n instanceof ArrayBuffer){t=n}}const i=c(e,t,o);i();return s}function c(n,t,o){var r=p;const s=function(){while(t.byteLength){if(n.bufferedAmount>n.bufferedAmountLowThreshold){n.onbufferedamountlow=()=>{n.onbufferedamountlow=null;s()}}if(t.byteLength<r){r=t.byteLength}const e=t.slice(0,r);t=t.slice(r);n.send(e);if(r!=p){n.send("");if(o)o();return}}};return s}async function l(u,h){const g=new TextEncoder;return new Promise(async(e,n)=>{for(const i of h.entries()){var t=`--${y}
`;const a=i[0];const c=i[1];if(typeof c==="string"){t+=`Content-Disposition: form-data; name="${a}"

`;u.send(g.encode(t));u.send(g.encode(c))}else{const l=c;t+=`Content-Disposition: form-data; name="${a}"; filename="${l.name}"
`;if(l.type){t+=`Content-Type: ${l.type}

`}else{t+="Content-Type: application/octet-stream\n\n"}u.send(g.encode(t));var o;var r=new Promise((e,n)=>{o=e});const f=new FileReader;var s=0;f.onerror=e=>{console.log("Error reading file",e)};f.onabort=e=>{console.log("File reading aborted",e)};f.onload=e=>{const n=e.target.result;u.send(n);s+=n.byteLength;if(s<l.size){d(s)}else{o()}};const d=e=>{f.readAsArrayBuffer(l.slice(e,e+p))};d(s);await r}}u.send(g.encode(`
--${y}--
`));u.send("");e()})}function v(e){const n=e.split("\n");const t={};for(const o of n){const r=o.search(":");if(r===-1){continue}const s=o.slice(0,r);const i=o.slice(r+1);t[s]=i.trim()}return t}function b(e){if(!e.startsWith("HTTP/1.1")){throw new Error(`unexpected status line: ${e}`)}const n=e.split(" ");if(n.length<3){throw new Error(`unexpected status line: ${e}`)}const t=parseInt(n[1]);const o=n.slice(2).join(" ");return{status:t,statusText:o}}function o(m){m.bufferedAmountLowThreshold=n;m.binaryType="arraybuffer";let e=(e,n,l)=>{var f=()=>{};var t=()=>{};const o=new Promise((e,n)=>{f=e;t=n});var d="";var u=-1;var h="";var g={};const p=new MessageChannel;var w=[];m.onmessage=o=>{if(o.data instanceof ArrayBuffer){if(u===-1){const r=d.slice(0,d.search("\n"));d=d.slice(d.search("\n")+1);const s=b(r);u=s.status;h=s.statusText;g=v(d);d="";w.push(o.data);p.port1.postMessage(null);const i=new Headers;for(const a in g){i.append(a,g[a])}if(i.has("Content-Length")&&l){const c=parseInt(i.get("Content-Length"));l(-1,c)}let e={status:u,statusText:h,headers:i};let n=new ReadableStream({type:"bytes",start(e){if(e instanceof ReadableByteStreamController){if(e.byobRequest){throw new Error("byobRequest not supported")}}},pull(r){return new Promise((o,e)=>{p.port2.onmessage=e=>{const n=w.shift();if(!n){m.send("");r.close();o();l==null?void 0:l(0);return}const t=n.byteLength;r.enqueue(new Uint8Array(n));o();l==null?void 0:l(t)}})}});let t=new Response(n,e);f(t)}else{const e=o.data;if(0<e.byteLength){w.push(e);p.port1.postMessage(null)}}}else if(typeof o.data==="string"){if(u===-1){d+=o.data}else{p.port1.postMessage(null)}}};if((n==null?void 0:n.body)instanceof FormData){if(!n.headers){n.headers=new Headers}if(n.headers instanceof Headers){n.headers.append("Content-Type","multipart/form-data; boundary="+y)}else if(typeof n.headers==="object"){if(n.headers instanceof Array){n.headers.push(["Content-Type","multipart/form-data; boundary="+y])}else{n.headers["Content-Type"]="multipart/form-data; boundary="+y}}}r(m,e,n).then(()=>{s(m,n==null?void 0:n.body).catch(e=>{t(e)})}).catch(e=>{t(e)});return o};return e}var t=class{constructor(e,n){this.answered=false;this.connectionPromiseResolve=()=>{};this.onAnswer=n;this.peerConnection=new RTCPeerConnection(e);this.connectionPromise=new Promise((e,n)=>{this.connectionPromiseResolve=e});this._configurePeerConnection()}_configurePeerConnection(){const t=this.peerConnection;t.onicegatheringstatechange=e=>{console.log("onicegatheringstatechange",t.iceGatheringState);let n=e.target;if(n.iceGatheringState==="complete"&&n.localDescription){this.onAnswer(n.localDescription)}};t.ondatachannel=e=>{console.log("ondatachannel",e);window.fetch=o(e.channel);window.rtcReady=true;this.connectionPromiseResolve()};t.onnegotiationneeded=e=>{console.log("onnegotiationneeded")};t.onsignalingstatechange=e=>{console.log("onsignalingstatechange",t.signalingState)};t.oniceconnectionstatechange=e=>{console.log("oniceconnectionstatechange",t.iceConnectionState)};t.onicecandidate=e=>{console.log("onicecandidate",e)}}async answerOffer(e){if(this.answered){return this.connectionPromise}const n=this.peerConnection;try{await n.setRemoteDescription(e);const t=await n.createAnswer();await n.setLocalDescription(t);this.answered=true}catch(e){console.error(e)}return this.connectionPromise}};function w(e){var n;if(e instanceof HTMLScriptElement){(n=e.parentNode)==null?void 0:n.replaceChild(i(e),e)}else{var t=-1,o=e.childNodes;while(++t<o.length){w(o[t])}}}function i(e){var n=document.createElement("script");n.text=e.innerHTML;var t=-1,o=e.attributes,r;while(++t<o.length){n.setAttribute((r=o[t]).name,r.value)}return n}function m(e,n){const t=document.createElement("a");t.setAttribute("style","display: none");document.body.appendChild(t);const o=new Blob([e],{type:"stream/octet"});const r=window.URL.createObjectURL(o);t.href=r;t.download=n;t.click();window.URL.revokeObjectURL(r)}var T=/^text\/.*$/;async function a(e,n){const o=document.createElement("span");document.body.innerHTML="";document.body.appendChild(o);var r=0;var s=0;const t=(e,n)=>{if(e===-1&&n){r=n}else if(0<e){s+=e;if(r){const t=(s/r*100).toFixed(2);o.innerText=`receiving data: ${t}%`}else{o.innerText=`receiving data: ${s} bytes`}}else{o.innerText=`receiving data: done`}return Promise.resolve()};const i=fetch;var a=await i(e,n,t);const c=a.headers;var l=c.get("Content-Type")?c.get("Content-Type"):"";l=l.split(";")[0];var f=c.get("Content-Disposition")?c.get("Content-Disposition"):"";if(!l){l="text/plain"}const d=C(f);if(d){const u=await a.blob();m(u,d);return}if(l.match(T)){const h=await a.text();if(l==="text/html"){const g=new DOMParser;const p=g.parseFromString(h,"text/html");document.body=p.body;w(document.body)}else{document.body.innerText=h;document.body.innerHTML=`<pre>${h}</pre>`}}else if(l.startsWith("application/")){const h=await a.blob();let e=new Blob([h],{type:l});let n=URL.createObjectURL(e);window.open(n,"_self")}else{console.log(`falling back to displaying body as preformatted text`);const h=await a.text();document.body.innerText=h;document.body.innerHTML=`<pre>${h}</pre>`}}function C(e){if(!e||!e.includes("attachment")){return""}const n=/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/;const t=n.exec(e);if(t!=null&&t[1]){return t[1].replace(/['"]/g,"")}return""}function f(n,t){return e=>{fetch(n,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({SessionID:t,Answer:e.sdp})}).then(e=>{if(e.status!==200){alert("failed to send answer: "+e.status+" "+e.statusText)}}).catch(e=>{alert(`failed to send answer: ${e}`)})}}function d(e){const n=JSON.stringify(e);const t=document.getElementById("answer-container");if(t){t.innerText=n}navigator.clipboard.writeText(n)}function u(e){if(!e.offer){alert("no offer");return}new t(e.rtcConfig,e.onAnswer).answerOffer(e.offer).then(()=>{a("/",{})})}window.WebRTCClient=t;function g(){const e=document.getElementById("connect-button");if(e){e.onclick=()=>{try{u(A(false))}catch(e){alert(e);return}}}else{const n=document.createElement("span");n.innerText="tunnelling to oneshot server...";document.body.appendChild(n);try{u(A(true))}catch(e){alert(e);return}}}function A(e){const n={};const t=window.config;if(!t){throw new Error("no config")}n.rtcConfig=JSON.parse(t.RTCConfigurationJSON);if(!n.rtcConfig){throw new Error("no rtc config")}if(!n.rtcConfig.iceServers){throw new Error("no ice servers")}n.sessionID=t.SessionID;if(!n.sessionID){throw new Error("no session id")}n.offer=JSON.parse(t.OfferJSON);if(!n.offer){throw new Error("no offer")}n.endpoint=t.Endpoint;if(!n.endpoint){throw new Error("no endpoint")}if(e){n.onAnswer=f(n.endpoint,n.sessionID)}else{n.onAnswer=d}return n}g()})();