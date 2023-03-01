import { sendFormData } from "./src/sendFormData";
import { sendString } from "./src/sendString";

function main() {
    console.log("running main")
    addFormSubmit();
    addStringSubmit();
}

function addFormSubmit() {
    const formEl = document.getElementById("file-form") as HTMLFormElement;
    if (!formEl) {
        return;
    }

    formEl.addEventListener("submit", (e) => {
        e.preventDefault();
        e.stopPropagation();

        const formData = new FormData(formEl);
        responsePromiseHandler(sendFormData(formData));
    });
}

function addStringSubmit() {
    const taEl = document.getElementById("text-input") as HTMLTextAreaElement;
    const formEl = document.getElementById("text-form") as HTMLFormElement;

    if (!taEl || !formEl) {
        return;
    }

    formEl.addEventListener("submit", (e) => {
        e.preventDefault();
        e.stopPropagation();

        responsePromiseHandler(sendString(taEl.value));
    });
}

function responsePromiseHandler(p: Promise<Response>) {
    p.then((response) => {
        if (response.ok) {
            console.log("Transfer succeeded");
            document.body.innerHTML = "Transfer succeeded";
        } else {
            const msg = "Transfer failed: " + response.status.toString + " " + response.statusText;
            console.log(msg);
            document.body.innerHTML = msg;
        }
    }).catch((err) => {
        if (err instanceof Error) {
            if (err.message === "cannot send empty data") {
                console.log(err.message);
                alert(err.message);
                return;
            }
        }
        const msg = "Transfer failed: " + err;
        console.log(msg)
        document.body.innerHTML = msg;
    });
}

main();