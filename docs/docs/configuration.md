# Configuration

There are plans to allow configuration via environment variables and config files in a future release.
For now, all config options are supplied via command line arguments / flags.

## Global Options

These options apply to both Server and CLI mode.

| Flag             | Default                   | Description                                                          |
|------------------|---------------------------|----------------------------------------------------------------------|
| `--debug`        | `false`                   | Display additional debug information, including all config values.   |
| `--help`/ `-h`   |                           | Display all commands and flags.                                      |
| `--log`/`-l`     | `false`                   | Save raw IRC logs for each client connection.                        |
| `--name`/`-n`    | **REQUIRED**              | Username used to connect to IRC server.                              |
| `--searchbot`    | `search`                  | The IRC search operator to use. Try `searchook` if `search` is down. |
| `--server`/`-s`  | `irc.irchighway.net:6697` | The IRC `server:port` to connect to.                                 |
| `--tls`          | `true`                    | Connect to IRC server over TLS.[^2]                                  |
| `--useragent/-u` | `OpenBooks v4.5.0`        | UserAgent / Version Reported to IRC Server.                          |

## Server Mode Options

| Flag                     | Default     | Description                                               |
|--------------------------|-------------|-----------------------------------------------------------|
| `--allowed-origins`      |             | Extra origins allowed to use the web UI.[^3]              |
| `--basepath`             | `/`         | Web UI Path. Must have trailing `/`. (Ex. `/openbooks/`)  |
| `--browser`/`-b`         | `false`     | Open the browser on startup.                              |
| `--dir`/`-d`             | `/temp`[^1] | Directory where search results and eBooks are saved.      |
| `--host`                 | All         | Interface address to listen on. (Ex. `127.0.0.1`)         |
| `--no-browser-downloads` | `false`     | Don't send files to browser but save them to disk.        |
| `--persist`              | `false`     | Save eBook files after sending to browser.                |
| `--port`/`-p`            | `5228`      | The port that the server listens on.                      |
| `--rate-limit`/`-r`      | `10`        | Seconds to wait between IRC search requests. (minimum 10) |

## CLI Mode Options

| Flag         | Default           | Description                                          |
|--------------|-------------------|------------------------------------------------------|
| `--dir`/`-d` | Working Directory | Directory where search results and eBooks are saved. |

[^1]: Docker sets a static directory of `/books` so that the volume is accessible outside the container.
[^2]: The server's TLS certificate is verified first. If verification fails (irc.irchighway.net currently uses a certificate that can't be verified), a warning is logged and OpenBooks falls back to an unverified TLS connection.
[^3]: The web UI only accepts websocket connections from pages served by OpenBooks itself, so other websites can't control it through your browser. The `Host` and `X-Forwarded-Host` headers are used to recognize your own address. Only use `--allowed-origins https://books.example.com` (comma separated) if your reverse proxy rewrites both headers.
