import * as esbuild from 'esbuild';
import fs from 'node:fs';
import UglifyJS from 'uglify-js';

let buildHTML = {
    name: 'build-html',
    setup(build) {
        build.onEnd(async result => {
            let mainJS = fs.readFileSync('./dist/main.js', 'utf8');
            mainJS = UglifyJS.minify(mainJS, {
                compress: false,
                mangle: true,
            }).code;
            fs.writeFileSync('./dist/main.minified.js', mainJS, 'utf8', (err) => {
                if (err) throw err;
            });
            fs.copyFileSync(
                './node_modules/@stardazed/streams-polyfill/dist/sd-streams-polyfill.min.js', 
                './dist/sd-streams-polyfill.min.js');
        });
    },
}

await esbuild.build({
    entryPoints: ['./main.ts'],
    bundle: true,
    outfile: './dist/main.js',
    target: ['chrome58', 'firefox57', 'safari11', 'edge16'],
    plugins: [buildHTML],
});