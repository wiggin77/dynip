# dynip

[![CI](https://github.com/wiggin77/dynip/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/wiggin77/dynip/actions/workflows/ci.yml)

Client for easyDNS Dynamic DNS service.

[easyDNS Dynamic DNS](https://kb.easydns.com/knowledge/dynamic-dns/)

"Dynamic DNS service is used to keep a domain name pointing to the same computer or server connected to the Internet despite the fact that the address (IP address) of the computer keeps changing. This service is useful to anyone who wants to operate a server (web server, mail server, ftp server, irc server etc) connected to the Internet with a dynamic IP or for someone who wants to connect to an office computer or server from a remote location with software such as pcAnywhere."

Prebuilt binaries available for Windows (amd64), macOS (amd64 and Apple Silicon arm64), Linux (amd64 and arm64) and OpenBSD (amd64) [here.](https://github.com/wiggin77/dynip/releases) Each release includes a `SHA256SUMS` file for verifying downloads.

## Usage

Dynip can be run manually as a command line utility, or installed to run as a service. In either case a config file must be created using the example [dynip.conf](./dynip.conf) as a template.

To update your IP address once:

```bash
dynip -f dynip.conf
```

Dynip will print one of the following messages. Success messages go to stdout, failures to stderr.

| MESSAGE | EXIT CODE | DESCRIPTION |
| ------- | --------- | ----------- |
| SUCCESS | 0 | IP address updated successfully
| NO_CHANGE | 0 | IP address is already the requested value
| TOO_SOON | 255 | the request was issued before minimum interval elapsed
| NO_SERVICE | 255 | Dynamic DNS is not turned on for this domain
| NO_AUTH | 255 | password/token incorrect
| ILLEGAL_INPUT | 255 | a request parameter was invalid
| NO_PARTNER | 255 | the request did not include required partner information
| SERVER_ERROR | 255 | a generic error occurred on the server
| UNKNOWN_RESPONSE | 255 | the server response contained no known result code
| LOCAL_ERROR: _reason_ | 255 | there was a local error, such as a missing config file or no network connection

On Windows the failure exit code is reported as -1 rather than 255. Invalid command line flags exit with code 2.

### Command line flags

| FLAG | DESCRIPTION |
| ---- | ----------- |
| `-f <file>` | config file (see [Config file location](#config-file-location)) |
| `-v` | verbose logging, overrides the `verbose` config setting (one-shot mode only) |
| `-i` | install as a service |
| `-u` | uninstall the service |
| `-user <name>` | with `-i`, the account the service runs as (Windows) |
| `-pw <password>` | with `-i`, the password for `-user` (Windows) |
| `-d` | run as a daemon; used by the service manager |
| `-version` | display version info |
| `-h` | display help |

### Config file location

If `-f` is not given, dynip looks for `~/.config/dynip/dynip.conf` in the current user's home directory, then falls back to `/etc/dynip.conf` if that file exists. When installing a service, pass `-f` explicitly; a relative path is converted to an absolute one at install time.

### Logging

By default dynip logs to stdout. Set `log` in the config file to append to a log file instead, or `syslog = YES` to log to syslog (Linux, *BSD, macOS). When `syslog` is enabled the `log` setting is ignored.

## Installation

### Linux

Dynip can install itself automatically to run as a service on most Linux systems using systemd, Upstart, SysV, OpenRC, procd or rc.d.

1. Copy dynip binary to `/usr/local/bin` or wherever you prefer to store your service binaries.

2. Create `/etc/dynip.conf` using the example [dynip.conf](./dynip.conf) as a template.

3. Run the commands:

      ```bash
      # this will install dynip as a service
      sudo /usr/local/bin/dynip -i -f /etc/dynip.conf

      # start the service
      sudo service dynip start

      # check service status
      sudo service dynip status
      ```

Alternatively you can configure Linux to run dynip as a service yourself. You must ensure the service manager starts dynip with the "-d" flag, and you may include "-f" specifying the location of dynip.conf (see [Config file location](#config-file-location)).

To uninstall:

```bash
sudo service dynip stop
sudo /usr/local/bin/dynip -u
```

### Windows 10 or newer

Dynip can install itself automatically to run as a service on Windows 10, Windows Server 2016 or later.

1. Copy dynip binary to `c:\dynip\dynip.exe` or wherever you prefer to store your service binaries.

2. Create `c:\dynip\dynip.conf` using the example [dynip.conf](./dynip.conf) as a template.

3. Press Windows+R to open the "Run" box. Type "cmd" into the box and then press Ctrl+Shift+Enter to open a command prompt as an administrator. Run the following command to install dynip as a service:

      ```bat
      c:\dynip\dynip.exe -i -f c:\dynip\dynip.conf
      ```

      To run the service as a specific user instead, pass the account name and password. Use `.\username` for a local account or `DOMAIN\username` for a domain account:

      ```bat
      c:\dynip\dynip.exe -i -f c:\dynip\dynip.conf -user .\username -pw password
      ```

4. Press Windows+R to open the "Run" box. Type "services.msc" into the box and then press enter to open Windows Services Manager. Ensure the Start-up Type for dynip is "Automatic". Right click on dynip and select "Start".

To uninstall, stop the dynip service in Windows Services Manager, then run from an administrator command prompt:

```bat
c:\dynip\dynip.exe -u
```

### macOS/Launchd

Not yet tested, but this should work on macOS 12 Monterey or later.

```bash
# this will install dynip as a service
sudo /usr/local/bin/dynip -i -f /etc/dynip.conf
```

### Other

One-shot mode (`dynip -f dynip.conf`) works on any platform supported by Go 1.26.3 or later. Follow the instructions below to build dynip for your platform.

Running as a service (`-i` and `-d`) requires a service manager that dynip supports: the Linux systems listed above, macOS launchd, Windows, FreeBSD, Solaris or AIX. On other platforms, such as OpenBSD, schedule one-shot updates instead, for example with cron. easyDNS rejects updates more often than every 10 minutes:

```
*/15 * * * * /usr/local/bin/dynip -f /etc/dynip.conf
```

## Building from source

1. Install [Go 1.26.3](https://golang.org/dl/) or newer.

2. Clone this repo to your local machine.

3. Run `go build`.

To cross-compile the release binaries (Linux and macOS amd64/arm64, Windows and OpenBSD amd64) into `./build/`, run `./build.sh`.
