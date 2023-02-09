export function triggerDownload(data: any, filename: string) {
    const a = document.createElement("a");
    a.setAttribute("style", "display: none");
    document.body.appendChild(a);

    const blob = new Blob([data], { type: "stream/octet" });
    const url = window.URL.createObjectURL(blob);
    a.href = url;
    a.download = filename;
    a.click();
    window.URL.revokeObjectURL(url);
}