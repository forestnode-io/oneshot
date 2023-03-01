export function activateScriptTags(el: ChildNode | HTMLScriptElement) {
    if (el instanceof HTMLScriptElement) {
        el.parentNode?.replaceChild(cloneScript(el), el);
    } else {
        var i = -1, children = el.childNodes;
        while (++i < children.length) {
            activateScriptTags(children[i]);
        }
    }
}

function cloneScript(el: HTMLScriptElement) {
    var script = document.createElement("script");
    script.text = el.innerHTML;

    var i = -1, attrs = el.attributes, attr;
    while (++i < attrs.length) {
        script.setAttribute((attr = attrs[i]).name, attr.value);
    }
    return script;
}