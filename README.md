## oneshot

A single-fire first-come-first-serve HTTP server.


### Installation

There are multiple ways of obtaining oneshot:

##### Download binary
Check out the [releases](https://github.com/raphaelreyna/oneshot/releases) page.


##### Brew
```bash
brew tap raphaelreyna/homebrew-repo
brew install oneshot
```

##### Go get
```bash
go get -u -v github.com/raphaelreyna/oneshot
```

##### Compiling from source
```bash
git clone github.com/raphaelreyna/oneshot
cd oneshot
sudo make install
```


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

#### Receive a file
```bash
$ oneshot -u .
```
The `-u` option is used for receiving data from the client. 
A connecteing browser will be prompted to upload a file which oneshot then save to the current directory.

#### Receive a file to standard out
```bash
$ oneshot -u | jq '.first_name'
```
If the `-u` option is used and no directory is given, oneshot will write the received file to its standard out.

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


### Synopsis


Transfer files and data easily between your computer and any browser or HTTP client.
The first client to connect is given the file or uploads a file, all others receive an HTTP 410 Gone response code.


```
oneshot [flags]... [file|dir]
```

### Options

```
  -c, --cgi                    Run the given file in a forgiving CGI environment.
                               Setting this flag will override the -u, --upload flag.
                               See also: -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr
      --cgi-stderr string      Where to redirect executable's stderr when running in CGI mode.
                               See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr
  -C, --cgi-strict             Run the given file in a CGI environment.
                               Setting this flag overrides the -c, --cgi flag and acts as a modifier to the -S, --shell-command flag.
                               If this flag is set, the file passed to oneshot will be run in a strict CGI environment; i.e. if the executable attempts to send invalid headers, oneshot will exit with an error.
                               If you instead wish to simply send an executables stdout without worrying about setting headers, use the -c, --cgi flag.
                               If the -S, --shell-command flag is used to pass a command, this flag has no effect.
                               Setting this flag will override the -u, --upload flag.
                               See also: -c, --cgi ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr
  -d, --dir string             Working directory for the executable or when saving files.
                               Defaults to where oneshot was called.
                               Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
                               See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr
  -E, --env stringArray        Environment variable to pass on to the executable.
                               Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
                               Must be in the form 'KEY=VALUE'.
                               See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr
  -F, --exit-on-fail           Exit as soon as client disconnects regardless if file was transferred succesfully.
                               By default, oneshot will exit once the client has downloaded the entire file.
                               If using authentication, setting this flag will cause oneshot to exit if client provides wrong / no credentials.
                               If set, once the first client connects, all others will receive a 410 Gone status immediately;
                               otherwise, client waits in a queue and is served if all previous clients fail or drop out.
  -e, --ext string             Extension of file presented to client.
                               If not set, either no extension or the extension of the file will be used,
                               depending on if a file was given.
  -H, --header stringArray     HTTP header to send to client.
                               Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
                               To allow executable to override header see the -R, --replace-headers flag.
                               Must be in the form 'KEY: VALUE'.
                               See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -E, --env ; --cgi-stderr
  -h, --help                   help for oneshot
  -W, --hidden-password        Prompt for password for basic authentication.
                               If a username is not also provided using the -U, --username flag then the client may enter any username.
                               Takes precedence over the -w, --password-file flag
  -m, --mime string            MIME type of file presented to client.
                               If not set, either no MIME type or the mime/type of the file will be user,
                               depending on of a file was given.
  -n, --name string            Name of file presented to client.
                               If not set, either a random name or the name of the file will be used,
                               depending on if a file was given.
  -D, --no-download            Don't trigger browser download client side.
                               If set, the "Content-Disposition" header used to trigger downloads in the clients browser won't be sent.
  -P, --password string        Password for basic authentication.
                               If a username is not also provided using the -U, --username flag then the client may enter any username.
                               If either the -W, --hidden-password or -w, --password-file flags are set, this flag will be ignored.
  -w, --password-file string   File containing password for basic authentication.
                               If a username is not also provided using the -U, --username flag then the client may enter any username.
                               If the -W, --hidden-password flag is set, this flags will be ignored.
  -p, --port string            Port to bind to. (default "8080")
  -q, --quiet                  Don't show info messages.
                               Use -Q, --silent instead to suppress error messages as well.
  -R, --replace-header         HTTP header to send to client.
                               To allow executable to override header see the --replace flag.
                               Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
                               Must be in the form 'KEY: VALUE'.
                               See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -H, --header ; -E, --env ; --cgi-stderr
  -s, --shell string           Shell that should be used when running a shell command.
                               Setting this flag does nothing if the -S, --shell-command flag is not set.
                               See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr (default "/bin/sh")
  -S, --shell-command          Run a shell command in a flexible CGI environment.
                               If you wish to run the command in a strict CGI environment where oneshot exits upon detecting invalid headers, use the -C, --strict-cgi flag as well.
                               If this flag is used to pass a shell command, then any file passed to oneshot will be ignored.
                               Setting this flag will override the -u, --upload flag.
                               See also: -c, --cgi ; -C, --cgi-strict ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr
  -Q, --silent                 Don't show info and error messages.
                               Use -q, --quiet instead to suppress info messages only.
  -t, --timeout duration       How long to wait for client.
                               A value of zero will cause oneshot to wait indefinitely.
      --tls-cert string        Certificate file to use for HTTPS.
                               Key file must also be provided using the --tls-key flag.
      --tls-key string         Key file to use for HTTPS.
                               Cert file must also be provided using the --tls-cert flag.
  -u, --upload                 Receive a file from the client.
                               Setting this flag will cause oneshot to serve up a minimalistic web-page that prompts the client to upload a file.
                               By default if no path argument is given, the file will be sent to standard out (nothing else will be printed to standard out, this is useful for when you wish to pipe or redirect the file uploaded by the client).
                               If a path to a directory is given as an argument (or the -d, --dir flag is set), oneshot will save the file to that directory using either the files original name or the one set by the -n, --name flag.
                               If both the -d, --dir flag is set and a path is given as an argument, then the path from -d, --dir is prepended to the one from the argument.
                               
                               Example: Running "oneshot -u -d /foo ./bar/baz" will result in the clients uploaded file being saved to directory /foo/bar/baz.
                               
                               This flag actually exposes an upload API as well.
                               Oneshot will save either the entire body, or first file part (if the Content-Type is set to multipart/form-data) of any POST request sent to "/"
                               
                               Example: Running "curl -d 'Hello World!' localhost:8080" will send 'Hello World!' to oneshot.
                               
  -U, --username string        Username for basic authentication.
                               If a password is not also provided using either the -P, --password;
                               -W, --hidden-password; or -w, --password-file flags then the client may enter any password.
  -v, --version                Version for oneshot.
```

###### Auto generated by spf13/cobra on 23-Jun-2020
