export function sendFormData(formData: FormData): Promise<Response> {
    const lengths = [];
    var count = 0;

    for (const pair of formData.entries()) {
        count++;
        const entry = pair[1];
        if (entry instanceof File) {
            const name = entry.name;
            const size = entry.size;

            lengths.push(name + "=" + size.toString());
        }
    }

    if (count === 0) {
        return Promise.reject(new Error("cannot send empty data"));
    }

    return fetch("/", {
        method: "POST",
        headers: {
            "X-Oneshot-Multipart-Content-Lengths": lengths.join(";"),
        },
        body: formData,
    })
}