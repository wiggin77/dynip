# dynip

[![Build Status](https://travis-ci.org/wiggin77/dynip.svg?branch=master)](https://travis-ci.org/wiggin77/dynip)

Client for easyDNS Dynamic DNS service.

[easyDNS Dynamic DNS](https://kb.easydns.com/knowledge/dynamic-dns/)

"Dynamic DNS service is used to keep a domain name pointing to the same computer or server connected to the Internet despite the fact that the address (IP address) of the computer keeps changing. This service is useful to anyone who wants to operate a server (web server, mail server, ftp server, irc server etc) connected to the Internet with a dynamic IP or for someone who wants to connect to an office computer or server from a remote location with software such as pcAnywhere."

Prebuilt binaries avaiable for Windows, Mac, Linux [here.](https://github.com/wiggin77/dynip/releases)

## Usage

Dynip can be run manually as a command line utility, or installed to run as a service. In either case a config file must be created using the example [dynip.conf](./dynip.conf) as a template.

To update your IP address once:

```bash
dynip -f dynip.conf
```

Dynip will respond with one of:

| MESSAGE | RETURN | DESCRIPTION |
| ------- | ------ |----------- |
| SUCCESS | 0 |IP address updated successfully
| NO_CHANGE | 0 |IP address is already the requested value
| TOO_SOON | -1 |the request was issued before minimum interval elapsed
| NO_SERVICE | -1 | Dynamic DNS is not turned on for this domain
| NO_AUTH | -1 |password/token incorrect
| SERVER_ERROR | -1 | a generic error occurred on the server
| LOCAL_ERROR | -1 | there was a local error

## Installation

### Linux

Dynip can install itself automatically to run as a service on most Linux systems using Systemd, Upstart or SysV.

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

Alternatively you can configure Linux to run dynip as a service yourself. You must ensure the service manager starts dynip with "-d" flags, and you may include the "-f" specifying the location of dynip.conf if not located in `/etc/dynip.conf`.

### Windows 7 or newer

Dynip can install itself automatically to run as a service on Windows 7 or later.

1. Copy dynip binary to `c:\dynip\dynip.exe` or wherever you prefer to store your service binaries.

2. Create `c:\dynip\dynip.conf` using the example [dynip.conf](./dynip.conf) as a template.

3. Press Windows+R to open the “Run” box. Type “cmd” into the box and then press Ctrl+Shift+Enter to run the following command as an administrator.

      ```powershell
      # this will install dynip as a service
      dynip.exe -i -f c:\dynip\dynip.conf
      ```

      or

      ```powershell
      # this will install dynip as a service to run as a specific user
      # `username` and `password` for actual credentials
      dynip.exe -i -f c:\dynip\dynip.conf -user username -pw password
      ```

4. Press Windows+R to open the “Run” box. Type “services.msc” into the box and then press enter to open Windows Services Manager. Ensure the Start-up Type for dynip is "Automatic". Right click on dynip and select "Start".

### OSX/Launchd

Not yet tested, but this should work on a relatively recent version of OSX.

```bash
# this will install dynip as a service
sudo dynip.exe -i -f /etc/dynip.conf
```

### Other

Dynip should work on any platform supported by Go 1.11.x or later. Follow the instructions below to build dynip for your platform. You can then install as a service manually

## Building from source

1. Install [Go 1.11.x](https://golang.org/dl/) or newer.

2. Clone this repo to your local machine.

3. Run `go build`.
