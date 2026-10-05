# [v4.6.0] - 2026-10-04

## Security
- DCC file offers are only accepted when sent directly to your nickname as a CTCP message. Previously anyone in `#ebooks` could make every connected client download a file.
- File names received over DCC or found inside archives can no longer write outside the download directory.
- Deleting a book from the library can no longer delete files outside the download directory.
- The web UI only accepts websocket connections from its own origin. Use `--allowed-origins` if your reverse proxy rewrites the `Host` header.
- Line breaks are stripped from search queries and download requests so they can't send additional IRC commands.
- The IRC server's TLS certificate is verified when possible. If verification fails, a warning is logged and an unverified connection is used.
- Desktop mode only listens on `127.0.0.1`. Server mode has a new `--host` flag.

## Fixed
- The server no longer crashes when the browser disconnects during a download, or when it receives a malformed websocket message or IRC notice.
- Correctly reply to IRC `PING` messages.
- Download servers at the beginning or end of a `NAMES` line were missing from the server list.
- Search results without a ` - ` author separator are no longer reported as parse errors.
- Wait for the IRC server to accept the connection instead of a fixed 2 second delay. A suffix is added to your nickname if it is already in use.
- DCC downloads time out instead of hanging forever and failed downloads no longer leave `.temp` files behind.
- The browser is notified when the IRC connection drops.

## Changed
- Requires Go 1.26 or newer. Dependencies updated. `mholt/archiver` replaced with `mholt/archives`.
- Docker images are built with Node 24 and Go 1.27.

# [v4.5.0] - 2023-01-08

## Added
-  Use `--useragent/-u` flag to optionally specify the [UserAgent](https://en.wikipedia.org/wiki/Client-to-client_protocol#VERSION) reported to the IRC server. Default remains `OpenBooks v4.5.0`.

## Breaking 
- `--name/-n` flag **must** be specified when starting the application. OpenBooks will no longer generate a random `noun_adjective` username.
- Only a single connection to the IRC server will be made. Opening a second browser tab will show an error message.


