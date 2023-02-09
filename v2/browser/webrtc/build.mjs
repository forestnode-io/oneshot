import * as esbuild from 'esbuild';
import fs from 'node:fs';
import UglifyJS from 'uglify-js';
import Mustache from 'mustache';

const htmlTemplate = `<!DOCTYPE html>
<html>
<head></head>
<body>
    <div id="answer-component">
        <label for="ice-server-url">ICE Server URL</label><br>
        <input type="text" id="ice-server-url" value="{{iceServerURL}}"><br>
        <br>
        <label for="server-sd">Oneshot Session Offer</label><br>
        <textarea id="server-sd" rows="10" cols="50"></textarea><br>
        <br>
        <label for="httpRequest">HTTP Request (optional)</label><br>
        <textarea id="httpRequest" rows="10" cols="50"></textarea><br>
        <br>
        <button type="button" id="connect-button">Generate Answer & Copy to Clipboard</button><br>
        <div id="answer-container">
            <span id="answer-span"></span>
        </div>
    </div>
</body>
<footer>
    <script>
    {{{main}}}
    </script>
</footer>
</html>`

let examplePlugin = {
    name: 'example',
    setup(build) {
        build.onEnd(async result => {
            let buf = fs.readFileSync('./dist/main.js', 'utf8');
            buf = UglifyJS.minify(buf, { compress: false,  mangle: {reserved: ["connect"]}}).code;
            buf = Mustache.render(htmlTemplate, { main: buf, iceServerURL: 'stun:stun.l.google.com:19302' });
            fs.writeFile('./dist/index.html', buf, 'utf8', (err) => { });
        });
    },
}

await esbuild.build({
    entryPoints: ['./src/connect.ts'],
    bundle: true,
    outfile: './dist/main.js',
    target: ['chrome58', 'firefox57', 'safari11', 'edge16'],
    plugins: [examplePlugin],
    metafile: true,
})
