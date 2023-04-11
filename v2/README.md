## oneshot v2

A single-fire, first-come-first-served HTTP server.
Send files to and from a terminal and any HTTP client including web browsers.
Oneshot supports not only sending files locally but also offers several methods for sending files across private networks. 
Rich JSON output is also supported, which provides stats on the file transfer and the HTTP requests received.

### Use Cases & Examples

#### Send a file
```bash
$ oneshot send path/to/file.txt
```
Then, from a browser (or any HTTP client) simply go to your computers I.P. address and the file download will be triggered.

#### Send a file securely
```bash
$ oneshot send -u username -W path/to/file.txt
```
The `-W` option will cause oneshot to prompt you for a password.
Oneshot also supports HTTPS, simply pass in the key and certificate using the `--tls-key` and `--tls-cert` flags.

#### Receive a file
```bash
$ oneshot receive .
```
The `receive` subcommand is used for receiving data from the client. 
A connecting browser will be prompted to upload a file which oneshot then save to the current directory.

#### Receive a file to standard out
```bash
$ oneshot receive | jq '.first_name'
```
If the receive subcommand is used and no directory is given, oneshot will write the received file to its standard out.

#### Serve up a first-come-first-serve web page
```bash
$ oneshot send -D my/web/page.html
```
The `-D` flag tells oneshot to not trigger a download client-side.

#### Send the results of a lengthy process
```bash
$ sudo apt update | oneshot send -n apt-update.txt
```
Oneshot can transfer from its standard input; by default files are given a random name.
The optional flag `-n` sets the name of the file.

#### Wait until someone provides credentials to start a process, then send its output
```bash
$ oneshot send -u foo -P password -c my_non-cgi_script.sh
```
Oneshot can run your scripts and programs in a CGI flexible CGI environment.
Even non-CGI executables may be used; oneshot will provide its own default headers or you can set your own using the `-H` flag.

#### Create a single-fire api in a single line
```bash
$ oneshot exec -- 'echo "Hello $(jq -r '.name')!"'
```
Here, the `exec` subcommand tells oneshot to run its input as a shell command in a flexible CGI environment.

In another terminal we can test our api:
```bash
$ curl -X POST -H 'Content-Type: application/json' -d '{"name": "world"}' localhost:8080
Hello World!
```

#### Receive a file, do work on it locally and send back the results
```bash
$ oneshot -u | gofmt | oneshot -J
```
The `-J` flag we are using here tells oneshot to only start serving HTTP once it has received an EOF from its stdin.
This allows us to create unix pipelines without needing to specify a different port for each instance of oneshot.
In this scenario, the user would upload or type in some Go code and upon hitting the back button (refresh wont work !) or going back to the original URL, the user will receive their formatted Go code.

### Reporting Bugs, Feature Requests & Contributing
Please report any bugs or issues [here](https://github.com/raphaelreyna/oneshot/issues).

I consider oneshot to be *nearly* feature complete; feature requests and contributions are welcome.