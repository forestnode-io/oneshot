import * as esbuild from 'esbuild';
import fs from 'node:fs';
import UglifyJS from 'uglify-js';
import Mustache from 'mustache';

let buildHTML = {
    name: 'build-html',
    setup(build) {
        build.onEnd(async result => {
            let tmplt = fs.readFileSync('./src/index.template.html', 'utf8');
            let buf = fs.readFileSync('./dist/main.js', 'utf8');
            buf = UglifyJS.minify(buf, { compress: false, mangle: true}).code;
            buf = Mustache.render(tmplt, { main: buf, iceServerURL: 'stun:stun.l.google.com:19302' });
            fs.writeFile('./dist/index.html', buf, 'utf8', (err) => { 
                if (err) throw err;
            });
            fs.unlink('./dist/main.js', (err) => {
                if (err) throw err;
            });
        });
    },
}

await esbuild.build({
    entryPoints: ['./src/connect.ts'],
    bundle: true,
    outfile: './dist/main.js',
    target: ['chrome58', 'firefox57', 'safari11', 'edge16'],
    plugins: [buildHTML],
});