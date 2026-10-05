# openbooks

> This is a maintained fork of [evan-buss/openbooks](https://github.com/evan-buss/openbooks), which hasn't had a release since 2023. It includes security fixes, crash fixes and updated dependencies. See the [changelog](docs/docs/changelog.md) for details.
>
> Only the latest release is supported. If you encounter any issues, be sure you are using the latest version.

[![Latest Release](https://img.shields.io/github/v/release/youenjoymyself/openbooks)](https://github.com/youenjoymyself/openbooks/releases/latest)
[![Test](https://github.com/youenjoymyself/openbooks/actions/workflows/test.yml/badge.svg)](https://github.com/youenjoymyself/openbooks/actions/workflows/test.yml)

Openbooks allows you to download ebooks from irc.irchighway.net quickly and easily.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./.github/home_v3_dark.png">
  <img alt="openbooks screenshot" src="./.github/home_v3.png">
</picture>

## Getting Started

Every mode requires `--name`, the nickname OpenBooks uses on IRC. If it's already taken, a suffix like `_1` is added.

### Binary

1. Download the latest release for your platform from the [releases page](https://github.com/youenjoymyself/openbooks/releases/latest).
   - Windows: `openbooks.exe`
   - macOS: `openbooks_mac_arm` (Apple Silicon) or `openbooks_mac` (Intel)
   - Linux: `openbooks_linux` (x86-64) or `openbooks_linux_arm` (ARM64)
2. On macOS and Linux, make it executable with `chmod +x [binary name]`. The macOS builds aren't signed, so also remove the download quarantine with `xattr -d com.apple.quarantine [binary name]`.
3. Run `./openbooks --name your_irc_nick`. This opens the web interface in your browser.
4. Run `./openbooks --help` to see all configuration options and the two other modes: CLI and Server.

### Docker

Images are published to the GitHub Container Registry for `linux/amd64`, `linux/arm64` and `linux/arm/v7`.

- Basic config
  - `docker run -p 8080:80 ghcr.io/youenjoymyself/openbooks --name your_irc_nick`
- Config to persist all eBook files to disk
  - `docker run -p 8080:80 -v ~/Downloads/openbooks:/books ghcr.io/youenjoymyself/openbooks --name your_irc_nick --persist`

Books are saved in a `books` subfolder of the mounted directory (`~/Downloads/openbooks/books` above). Without `--persist`, each book is deleted from the server after your browser downloads it. Add `--no-browser-downloads` to only save books to disk.

#### Docker Compose

```yaml
services:
  openbooks:
    image: ghcr.io/youenjoymyself/openbooks:latest
    container_name: openbooks
    restart: unless-stopped
    ports:
      - "8080:80"
    volumes:
      - ~/Downloads/openbooks:/books
    command: --name your_irc_nick --persist
```

On Unraid, add `user: "99:100"` so files on your shares are owned by `nobody:users` instead of root.

### Setting the Base Path

OpenBooks server doesn't have to be hosted at the root of your webserver. The basepath value allows you to host it behind a reverse proxy. The base path value must have opening and closing forward slashes (default "/").

- Docker
  - `docker run -p 8080:80 -e BASE_PATH=/openbooks/ ghcr.io/youenjoymyself/openbooks --name your_irc_nick`
- Binary
  - `./openbooks server --name your_irc_nick --basepath /openbooks/`

### Reverse Proxies

The web interface only accepts connections from its own address, so other websites can't control OpenBooks through your browser. OpenBooks recognizes its address from the `Host` and `X-Forwarded-Host` headers, which most reverse proxies pass through. If search never connects behind your proxy, add `--allowed-origins https://your.domain`.

## Usage

For a complete list of features use the `--help` flags on all subcommands.
For example `openbooks cli --help` or `openbooks cli download --help`.

- **Desktop** (default, `openbooks --name nick`): runs on this computer only and opens the web interface in your browser. Books are saved in a `books` folder inside your Downloads folder.
- **Server** (`openbooks server --name nick`): runs as a web application that you can visit in your browser, including from other devices on your network. This is what the Docker image runs.
- **CLI** (`openbooks cli --name nick`): search and download books from a terminal.

See the [configuration docs](docs/docs/configuration.md) for every flag.

## Development

Requires Go 1.26+ and Node.js. [Task](https://taskfile.dev) is optional but runs the common workflows.

### Build

The Go server embeds the React app, so build the frontend first:

```bash
cd server/app && npm ci && npm run build && cd ../..
go build -o openbooks ./cmd/openbooks
```

Run `task release:build` (or `./build.sh`) to compile binaries for every platform into `build/`.

### Test

```bash
go vet ./...
go test -race ./...
```

### Mock Development Server

The mock server allows you to debug responses and requests to simplified IRC / DCC
servers that mimic the responses received from IRC Highway.

```bash
task dev:mock
# Another Terminal
task dev:server        # or dev:cli. Set the nickname with NAME=<nick>
```

Without Task:

```bash
cd cmd/mock_server && go run .
# Another Terminal
go run ./cmd/openbooks server --tls=false --server localhost:6667 --name dev
```

Run `task dev:client` for the React development server with hot reloading.

### Desktop App

Compile OpenBooks with experimental webview support (requires cgo):

```shell
cd cmd/openbooks
go build -tags webview
```

## Why / How

From the original author, [Evan Buss](https://github.com/evan-buss):

- I wrote this as an easier way to search and download books from irchighway.net. It handles all the extraction and data processing for you. You just have to click the book you want. Hopefully you find it much easier than the IRC interface.
- It was also interesting to learn how the [IRC](https://en.wikipedia.org/wiki/Internet_Relay_Chat) and [DCC](https://en.wikipedia.org/wiki/Direct_Client-to-Client) protocols work and write custom implementations.

## Technology

- Backend
  - Golang
  - Chi
  - gorilla/websocket
  - mholt/archives (extract files from various archive formats)
- Frontend
  - React.js
  - TypeScript
  - Redux / Redux Toolkit
  - Mantine UI / @emotion/react
  - Framer Motion
