<img src="https://github.com/raphaelreyna/oneshot/raw/master/oneshot_banner.png" width="744px" height="384px">

![GitHub](https://img.shields.io/github/license/raphaelreyna/oneshot) ![GitHub release (latest by date)](https://img.shields.io/github/v/release/raphaelreyna/oneshot) [![Go Report Card](https://goreportcard.com/badge/github.com/raphaelreyna/oneshot)](https://goreportcard.com/report/github.com/raphaelreyna/oneshot) ![Twitter URL](https://img.shields.io/twitter/url?style=social&url=https%3A%2F%2Frphlrn.com%2Foneshot-site%2F)
## oneshot

A single-fire first-come-first-serve HTTP server.


#### A video overview of oneshot (thanks to Brodie Robertson)
<a href="https://www.youtube.com/watch?v=ZOHvdMgplz4">
  <img src="https://img.youtube.com/vi/ZOHvdMgplz4/maxresdefault.jpg" height="150px"/>
</a>


### Installation

There are multiple ways of obtaining oneshot:

#### Linux / macOS
Copy and paste any of these commands into your terminal to install oneshot.
For some portion of Linux users, there are .deb and .rpm packages available in the [release page](https://github.com/raphaelreyna/oneshot/releases).

##### Download binary (easiest)
```bash
curl -L https://github.com/raphaelreyna/oneshot/raw/master/install.sh | sudo bash
```

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

#### Arch Linux users
<a href="https://aur.archlinux.org/packages/oneshot/">Oneshot AUR page.</a>


#### Windows

##### Download executable
Head over to the [release page](https://github.com/raphaelreyna/oneshot/releases) and download the windows .zip file.

##### Go get
```powershell
go get -u -v github.com/raphaelreyna/oneshot
```


### Windows GUI
Windows users might be interested in checkout [Goneshot](https://github.com/raphaelreyna/goneshot)(beta).
If for some reason, you would rather *not* use a command line, [Goneshot](https://github.com/raphaelreyna/goneshot)(beta) wraps oneshot with a GUI (graphical user interface) which might be easier to use. A macOS version will probably be made at some point.


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
A connecting browser will be prompted to upload a file which oneshot then save to the current directory.

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
$ oneshot -U "" -P password -c my_non-cgi_script.sh
```
Oneshot can run your scripts and programs in a CGI flexible CGI environment.
Even non-CGI executables may be used; oneshot will provide its own default headers or you can set your own using the `-H` flag.
Passing in an empty value (`""`) for `-U, --username` or `-P, --password` will result in a randomly generate username or password.

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


### Synopsis


Transfer files and data easily between your computer and any browser or HTTP client.
The first client to connect is given the file or uploads a file, all others receive an HTTP 410 Gone response code.
Directories will automatically be archived before being sent (see -a, --archive-method for more information).


```
oneshot [flags]... [file|dir|url]
```

### Options

```
  -B, --allow-bots              Allow bots to attempt download.
                                By default, bots are prevented from attempting the download; this is required to allow links to be sent over services that provide previews such as Apple iMessage.
                                A client is considered to be a bot if the 'User-Agent' header contains either 'bot', 'Bot' or 'facebookexternalhit'.
                                
  -a, --archive-method string   Which archive method to use when sending directories.
                                Recognized values are "zip" and "tar.gz", any unrecognized values will default to "tar.gz". (default "tar.gz")
  -c, --cgi                     Run the given file in a forgiving CGI environment.
                                Setting this flag will override the -u, --upload flag.
                                See also: -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr
      --cgi-stderr string       Where to redirect executable's stderr when running in CGI mode.
                                See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr
  -C, --cgi-strict              Run the given file in a CGI environment.
                                Setting this flag overrides the -c, --cgi flag and acts as a modifier to the -S, --shell-command flag.
                                If this flag is set, the file passed to oneshot will be run in a strict CGI environment; i.e. if the executable attempts to send invalid headers, oneshot will exit with an error.
                                If you instead wish to simply send an executables stdout without worrying about setting headers, use the -c, --cgi flag.
                                If the -S, --shell-command flag is used to pass a command, this flag has no effect.
                                Setting this flag will override the -u, --upload flag.
                                See also: -c, --cgi ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr
  -d, --dir string              Working directory for the executable or when saving files.
                                Defaults to where oneshot was called.
                                Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
                                See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr
  -E, --env stringArray         Environment variable to pass on to the executable.
                                Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
                                Must be in the form 'KEY=VALUE'.
                                See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -H, --header ; --cgi-stderr
  -F, --exit-on-fail            Exit as soon as client disconnects regardless if file was transferred succesfully.
                                By default, oneshot will exit once the client has downloaded the entire file.
                                If using authentication, setting this flag will cause oneshot to exit if client provides wrong / no credentials.
                                If set, once the first client connects, all others will receive a 410 Gone status immediately;
                                otherwise, client waits in a queue and is served if all previous clients fail or drop out.
  -e, --ext string              Extension of file presented to client.
                                If not set, either no extension or the extension of the file will be used,
                                depending on if a file was given.
  -H, --header stringArray      HTTP header to send to client.
                                Setting a value for 'Content-Type' will override the -M, --mime flag.
                                To allow executable to override header see the -R, --replace-headers flag.
                                Must be in the form 'KEY: VALUE'.
                                See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -R, --replace-headers ; -E, --env ; --cgi-stderr
  -h, --help                    help for oneshot
  -W, --hidden-password         Prompt for password for basic authentication.
                                If a username is not also provided using the -U, --username flag then the client may enter any username.
                                Takes precedence over the -w, --password-file flag
  -M, --mdns                    Register oneshot as an mDNS (bonjour/avahi) service.
  -m, --mime string             MIME type of file presented to client.
                                If not set, either no MIME type or the mime/type of the file will be user,
                                depending on of a file was given.
  -n, --name string             Name of file presented to client.
                                If not set, either a random name or the name of the file will be used,
                                depending on if a file was given.
  -D, --no-download             Don't trigger browser download client side.
                                If set, the "Content-Disposition" header used to trigger downloads in the clients browser won't be sent.
  -L, --no-unix-eol-norm        Don't normalize end-of-line chars to unix style on user input.
                                Most browsers send DOS style (CR+LF) end-of-line characters when submitting user form input; setting this flag to true prevents oneshot from doing the replacement CR+LF -> LF.
                                This flag does nothing if both the -u, --upload and --upload-input flags are not set.
                                See also: -u, --upload; --upload-input
  -P, --password string         Password for basic authentication.
                                If an empty password ("") is set then a random secure will be used.
                                If a username is not also provided using the -U, --username flag then the client may enter any username.
                                If either the -W, --hidden-password or -w, --password-file flags are set, this flag will be ignored.
  -w, --password-file string    File containing password for basic authentication.
                                If a username is not also provided using the -U, --username flag then the client may enter any username.
                                If the -W, --hidden-password flag is set, this flags will be ignored.
  -p, --port string             Port to bind to. (default "8080")
  -q, --quiet                   Don't show info messages.
                                Use -Q, --silent instead to suppress error messages as well.
  -r, --redirect                Redirect the first client to connect to the URL given as the first argument to oneshot.
                                See also: --status-code
  -R, --replace-header          Allow executable to override headers set by  the -H, --header flag.
                                Setting this flag does nothing unless either the -c, --cgi or -S, --shell-command flag is set.
                                See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -S, --shell ; -H, --header ; -E, --env ; --cgi-stderr
  -s, --shell string            Shell that should be used when running a shell command.
                                Setting this flag does nothing if the -S, --shell-command flag is not set.
                                See also: -c, --cgi ; -C, --cgi-strict ; -s, --shell-command ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr (default "/bin/sh")
  -S, --shell-command           Run a shell command in a flexible CGI environment.
                                If you wish to run the command in a strict CGI environment where oneshot exits upon detecting invalid headers, use the -C, --strict-cgi flag as well.
                                If this flag is used to pass a shell command, then any file passed to oneshot will be ignored.
                                Setting this flag will override the -u, --upload flag.
                                See also: -c, --cgi ; -C, --cgi-strict ; -S, --shell ; -R, --replace-headers ; -H, --header ; -E, --env ; --cgi-stderr
  -Q, --silent                  Don't show info and error messages.
                                Use -q, --quiet instead to suppress info messages only.
  -T, --ss-tls                  Generate and use a self-signed TLS certificate/key pair for HTTPS.
                                A new certificate/key pair is generated for each running instance of oneshot.
                                To use your own certificate/key pair, use the --tls-cert and --tls-key flags.
                                See also: --tls-key ; -T, --ss-tls
      --status-code int         Sets the HTTP response status code when performing a redirect.
                                This flag does nothing if not redirecting to a different URL.
                                See also: -r, --redirect (default 303)
  -t, --timeout duration        How long to wait for client.
                                A value of zero will cause oneshot to wait indefinitely.
      --tls-cert string         Certificate file to use for HTTPS.
                                If the empty string ("") is passed to both this flag and --tls-key, then oneshot will generate, self-sign and use a TLS certificate/key pair.
                                Key file must also be provided using the --tls-key flag.
                                See also: --tls-key ; -T, --ss-tls
      --tls-key string          Key file to use for HTTPS.
                                If the empty string ("") is passed to both this flag and --tls-cert, then oneshot will generate, self-sign and use a TLS certificate/key pair.
                                Cert file must also be provided using the --tls-cert flag.
                                See also: --tls-cert ; -T, --ss-tls
  -u, --upload                  Receive a file, allow client to send text or upload a file to your computer.
                                Setting this flag will cause oneshot to serve up a minimalistic web-page that prompts the client to either upload a file or enter text.
                                To only allow for a file or user input and not both, see the --upload-file and --upload-input flags.
                                By default if no path argument is given, the file will be sent to standard out (nothing else will be printed to standard out, this is useful for when you wish to pipe or redirect the file uploaded by the client).
                                If a path to a directory is given as an argument (or the -d, --dir flag is set), oneshot will save the file to that directory using either the files original name or the one set by the -n, --name flag.
                                If both the -d, --dir flag is set and a path is given as an argument, then the path from -d, --dir is prepended to the one from the argument.
                                See also: --upload-file; --upload-input; -L, --no-unix-eol-norm
                                
                                Example: Running "oneshot -u -d /foo ./bar/baz" will result in the clients uploaded file being saved to directory /foo/bar/baz.
                                
                                This flag actually exposes an upload API as well.
                                Oneshot will save either the entire body, or first file part (if the Content-Type is set to multipart/form-data) of any POST request sent to "/"
                                
                                Example: Running "curl -d 'Hello World!' localhost:8080" will send 'Hello World!' to oneshot.
                                
      --upload-file             Receive a file, allow client to upload a file to your computer.
                                Setting both this flag and --upload-input is equivalent to setting the -u, --upload flag.
                                For more information see the -u, --upload flag documentation.
                                See also: --upload-input; -u, --upload
      --upload-input            Receive text from a browser.
                                Setting both this flag and --upload-file is equivalent to setting the -u, --upload flag.
                                For more information see the -u, --upload flag documentation.
                                See also: --upload-file; -u, --upload; -L, --no-unix-eol-norm
  -U, --username string         Username for basic authentication.
                                If an empty username ("") is set then a random, easy to remember username will be used.
                                If a password is not also provided using either the -P, --password flag ; -W, --hidden-password; or -w, --password-file flags then the client may enter any password.
  -v, --version                 Version and other info.
  -J, --wait-for-eof            Wait for EOF before starting HTTP(S) server if serving from stdin.
                                This flag does noting if not serving from stdin.
                                
```

###### Auto generated by spf13/cobra on 31-Oct-2020
