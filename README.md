# remote-open

Open URLs from a Linux server in your Mac browser. Works over SSH + WebSocket.

`xdg-open` on Linux → your Mac browser. External URLs open directly. Localhost URLs get an automatic SSH tunnel.

## Architecture

```
Linux                          Mac
─────                          ───
xdg-open → POST :20080  ───→  WS → open <url>
                          or   WS → ssh -L → open 127.0.0.1:<port>
```

## Setup

### 1. Server (Linux)

```bash
git clone https://github.com/nicco/remote-open.git
cd remote-open
docker build -t remote-open-server .
docker run -d --network host --name remote-open-server --restart unless-stopped remote-open-server --port 20080

# Install the xdg-open shim
GOOS=linux go build -o /usr/local/bin/remote-open-shim ./cmd/shim/
sudo chmod +x /usr/local/bin/remote-open-shim
sudo ln -sf /usr/local/bin/remote-open-shim /usr/bin/xdg-open

# Configure
mkdir -p ~/.remote-open
echo '{"server":"http://localhost:20080"}' > ~/.remote-open/config.json
```

### 2. Client (Mac)

```bash
git clone https://github.com/nicco/remote-open.git
cd remote-open
GOOS=darwin go build -o /usr/local/bin/remote-open-client ./cmd/client/

# Configure (replace IP with your server's IP)
mkdir -p ~/.remote-open
cat > ~/.remote-open/config.json << 'EOF'
{"server": "ws://YOUR_SERVER_IP:20080/ws", "ssh_user": "your_ssh_user"}
EOF

# Run
/usr/local/bin/remote-open-client
```

For persistence, add to `~/Library/LaunchAgents/io.github.remote-open-client.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>io.github.remote-open-client</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/remote-open-client</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/io.github.remote-open-client.plist
```

### 3. Requirements

- SSH access from Mac to Linux (key-based auth recommended)
- Port 20080 accessible from Mac to Linux

## How it works

1. Any tool on Linux calls `xdg-open <url>`
2. The shim POSTs the URL to the server on port 20080
3. The server pushes the URL to the connected Mac client via WebSocket
4. External URLs (`https://...`) → `open <url>` in Mac browser
5. Localhost URLs (`http://localhost:3000`) → `ssh -L 3000:localhost:3000` + `open http://127.0.0.1:3000`

## Config

`~/.remote-open/config.json`:

| Key | Required | Example |
|---|---|---|
| `server` | yes | `"ws://10.0.0.5:20080/ws"` (Mac) / `"http://localhost:20080"` (Linux) |
| `ssh_user` | Mac only | `"your_ssh_username"` |
