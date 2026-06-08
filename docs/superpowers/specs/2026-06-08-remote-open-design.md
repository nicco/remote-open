# remote-open Design

Forward URLs from a Linux server to a macOS client. `xdg-open` on Linux → Docker server → WebSocket → Mac client → macOS `open` (external URLs) or tunneled localhost (internal URLs).

## Architecture

Three components:

```
┌─────────────────┐     HTTP POST      ┌──────────────────────┐     WebSocket      ┌─────────────────────┐
│  xdg-open shim  │ ──────────────────> │  remote-open-server  │ <─────────────────> │  remote-open-client  │
│  (Linux host)   │   fire-and-forget   │  (Docker container)  │   persistent conn   │  (macOS daemon)     │
└─────────────────┘                     └──────────────────────┘                     └─────────────────────┘
                                                 │                                            │
                                                 │  TCP proxy via WS                          │  local TCP listeners
                                                 ▼                                            ▼
                                        ┌──────────────────┐                          ┌──────────────┐
                                        │ localhost:XXXX    │                          │ 127.0.0.1:XXXX │
                                        │ (dev servers etc) │                          │ (Mac browser)  │
                                        └──────────────────┘                          └──────────────┘
```

- **xdg-open shim**: Go binary on the Linux host. Replaces `/usr/bin/xdg-open` via symlink. Receives a URL, POSTs it to the server, exits immediately.
- **remote-open-server**: Go binary in a Docker container. Single port `20080` serves both HTTP (`POST /open` for the shim) and WebSocket (`GET /ws` for the client). Proxies TCP traffic for localhost tunnels.
- **remote-open-client**: Go background daemon for macOS. Connects WebSocket to server. Opens external URLs via `open <url>`. For localhost URLs, starts a temporary TCP listener on the mirrored port and pipes traffic over the WebSocket to the Linux service.

All three components are configured with the same server address and port.

## Wire Protocol (WebSocket)

All messages are JSON with a `type` field. The same WebSocket carries control messages and tunneled TCP data.

### Message Types

| Type | Direction | Purpose |
|---|---|---|
| `open-url` | Server → Client | Push a URL from xdg-open to the client |
| `ping` | Client → Server | Health-check a tunneled port |
| `pong` | Server → Client | Response to ping |
| `proxy-data` | Bidirectional | Tunneled TCP traffic for a port |

### open-url

```
{ "type": "open-url", "url": "http://localhost:3000/path" }
```

The `url` is the raw string from xdg-open. The client parses it to decide: if host is `localhost` or `127.0.0.1`, extract port and start a tunnel; otherwise call `open <url>`.

### ping / pong

```
{ "type": "ping", "port": 3000 }
{ "type": "pong", "port": 3000, "alive": true }
```

Client sends `ping` periodically (~5s) for each active tunnel. Server does a TCP dial to `localhost:<port>` and responds with `alive`.

### proxy-data

```
{ "type": "proxy-data", "port": 3000, "data": "<base64>" }
```

Carries raw TCP bytes for a tunnel in either direction. The server unpacks and forwards to the Linux localhost service; the client unpacks and forwards to the Mac browser's TCP connection.

## Component Details

### xdg-open Shim

- Location: `/usr/local/bin/remote-open-shim`
- Replaces `/usr/bin/xdg-open` via symlink
- Configured via env var `REMOTE_OPEN_SERVER` or config file `/etc/remote-open/config`
- On invocation with a URL: `POST <server>/open` with the URL as plain text body
- Fire-and-forget — exits immediately, does not read the response
- If no URL argument: exits 0 silently
- If server unreachable: logs to stderr, exits 0

### remote-open-server

- Go binary, Dockerized (multi-stage build, final image `alpine` or `scratch`)
- Single port `20080` (configurable via `--port`)
- `POST /open` handler: broadcasts `{"type":"open-url","url":"..."}` to all connected WebSocket clients. Returns 200 regardless of client count.
- `GET /ws` handler: upgrades to WebSocket, manages client connections
- Internal state: list of connected clients (map of websocket.Conn)
- Ping handler: TCP dial to `localhost:<port>`, respond with alive status
- Proxy-data handler: unpack base64, dial Linux localhost, write, read response, pack back
- Runs with `--network host` in Docker to reach host localhost services directly

### remote-open-client

- Go background daemon for macOS, no UI
- Configured via `--server ws://<host>:20080/ws` or config file `~/.remote-open-client.json`
- On startup: connect WebSocket, retry with exponential backoff (1s → 30s max) on failure
- Message loop:
  - `open-url`: parse URL. If localhost/127.0.0.1 → start mirrored TCP listener on `127.0.0.1:<port>`, begin periodic ping loop. Otherwise → `exec("open", url)`.
  - `pong`: update tunnel liveness. If `alive:false` → increment miss counter. After 3 consecutive misses → tear down listener for that port.
  - `proxy-data`: forward `data` to the local TCP connection for `port`.
- Tunnel lifecycle: tied to the Linux service, not browser tabs. Pings succeed → tunnel stays up. Pings fail consistently → tunnel torn down.
- Port conflict: if `127.0.0.1:<port>` already in use, log and skip the tunnel. Still call `open` so the browser at least opens.
- Connection lost: retry WebSocket with backoff. While disconnected, URLs are lost (server fires and forgets).

## Deployment

### Server

```bash
docker run -d \
  --name remote-open-server \
  --network host \
  ghcr.io/nicco/remote-open-server:latest \
  --port 20080
```

### Shim

```bash
sudo mv /usr/bin/xdg-open /usr/bin/xdg-open.real
sudo ln -s /usr/local/bin/remote-open-shim /usr/bin/xdg-open
export REMOTE_OPEN_SERVER=http://<host-ip>:20080
```

### Client

```bash
remote-open-client --server ws://<host-ip>:20080/ws
```

Run via `launchd` for persistence:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.nicco.remote-open-client</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/remote-open-client</string>
        <string>--server</string>
        <string>ws://192.168.1.50:20080/ws</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
```

## Configuration Summary

| Piece | Config |
|---|---|
| Server | `--port 20080` |
| Shim | `REMOTE_OPEN_SERVER=http://<host>:20080` |
| Client | `--server ws://<host>:20080/ws` |

The server address and port are the only shared configuration.

## Error Handling

| Scenario | Behavior |
|---|---|
| Shim: server unreachable | Log to stderr, exit 0 |
| Shim: no URL argument | Exit 0 |
| Server: no clients connected | `/open` returns 200, URL dropped |
| Server: client disconnects | Clean up that client's tunnels |
| Server: localhost service unreachable (proxy) | Unpack fails, no data forwarded |
| Client: WebSocket lost | Retry with backoff (1s, 2s, 4s... max 30s) |
| Client: port conflict on Mac | Log, skip tunnel, still call `open` |
| Client: 3 consecutive ping misses | Tear down tunnel listener |
| Client: browser tab closes | TCP connection closes naturally; tunnel stays up |

## What's Out of Scope

- Authentication / security between components (assumes trusted LAN)
- Multiple server support (one server address)
- Queueing URLs when client is disconnected
- TLS encryption
- Windows or Linux client support (macOS only)
- UI (pure background daemon)
