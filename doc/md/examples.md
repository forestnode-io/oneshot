### Use Cases & Examples

#### Send a file
```bash
$ oneshot path/to/file.txt
```
Then, from a browser (or any HTTP client) simply go to your computers I.P. address and the file download will be triggered.

#### Send a file securely
```bash
$ oneshot -U username -W path/to/file.txt
```
The `-W` option will cause oneshot to prompt you for a password.
Oneshot also supports HTTPS, simply pass in the key and certificate using the `--tls-key` and `--tls-cert` flags.

#### Serve up a first-come-first-serve web page
```bash
$ oneshot -D my/web/page.html
```
The `-D` flag tells oneshot to not trigger a download client-side.

#### Send the results of a lengthy process
```bash
$ sudo apt update | oneshot -n apt-update.txt
```
Oneshot can transfer from its standard input; by default files are given a random name.
The optional flag `-n` sets the name of the file.

#### Wait until someone provides credentials to start a process, then send its output
```bash
$ oneshot -U username -P password -c my_non-cgi_script.sh
```
Oneshot can run your scripts and programs in a CGI flexible CGI environment.
Even non-CGI executables may be used; oneshot will provide its own default headers or you can set your own using the `-H` flag.

#### Create a single-fire api in a single line
```bash
$ oneshot -D -S 'echo "hello $(jq -r '.name')!"'
```
Here, the `-S` flag tells oneshot to run its input as a shell command in a flexible CGI environment.

#### Create a 3-way transaction
```bash
$ oneshot -D -S 'oneshot -p 8081 some_asset.mp3' 
```
In this scenario, Alice runs oneshot, Bob connects to Alice's machine and his browser hangs until Carol also connects; Bob then receives the mp3 file.
