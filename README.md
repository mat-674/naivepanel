# NaivePanel

**NaivePanel** is a lightweight web management panel for NaiveProxy (Caddy with forwardproxy plugin). It provides a clean dark-themed interface to manage proxy users, monitor traffic, and configure server settings.

## NaivePanel Dashboard
![NaivePanel Dashboard](https://raw.githubusercontent.com/mat-674/naivepanel/main/assets/preview.png)

## 🚀 Features

- **One-Command Installation**: Interactive setup wizard — prompts for domain, TLS email, admin and proxy credentials (all auto-generated if left empty). Builds everything from source.
- **Modern UI**: Dark-themed SPA with glassmorphism aesthetics.
- **Dynamic Caddyfile**: Automatically regenerates and reloads Caddy config on user create/update/delete.
- **Multi-User Support**: Create, update, and delete proxy users with per-user traffic limits and enable/disable toggles.
- **Private Admin Surface**: The panel binds to loopback only; Caddy exposes the public subscription endpoint over HTTPS.
- **Separate Services**: NaiveProxy and NaivePanel run as distinct systemd services, so proxy lifecycle survives panel restarts.
- **ACME / Let's Encrypt**: Built-in SSL certificate management via Caddy.
- **Connection Links & QR Codes**: Generates `naive+https://` URIs with QR codes on demand.
- **Service Control**: Start / Stop / Restart NaiveProxy directly from the panel UI.
- **Single Binary**: Written in Go; static frontend assets are embedded via `embed.FS`.

> Traffic limits, expiry, and HWID values are currently delivered as subscription metadata. Server-side traffic accounting and proxy-side quota enforcement are not implemented yet, so do not rely on those fields for access control.

## 🛠 Tech Stack

| Layer | Technology |
|---|---|
| **Backend** | Go 1.21+, `net/http`, SQLite (`go-sqlite3`), JWT (`golang-jwt/jwt v5`), bcrypt (`golang.org/x/crypto`) |
| **Frontend** | Vanilla HTML / CSS / JS — SPA, no build step required |
| **QR Codes** | `skip2/go-qrcode` — base64-encoded PNG served via API |
| **Proxy Core** | NaiveProxy (Caddy v2.9.1 + `klzgrad/forwardproxy`) |

## 📁 Project Structure

```
naivepanel/
├── main.go                    # Entry point, routing, CLI flags
├── go.mod / go.sum
├── internal/
│   ├── auth/                  # JWT generation & bcrypt helpers
│   ├── config/                # Config loading, random credential generators
│   ├── database/              # SQLite layer (admin, users, settings)
│   ├── handlers/              # HTTP handlers: auth, users, settings, status
│   ├── models/                # Shared structs (ProxyUser, Settings, etc.)
│   └── naiveproxy/            # Caddyfile generator, process manager, sysinfo
├── web/                       # Embedded frontend (SPA)
│   ├── index.html
│   ├── css/
│   └── js/
│       ├── api.js             # Fetch wrapper for REST API
│       ├── app.js             # Router & auth state
│       ├── components/        # Shared UI components
│       └── pages/             # login, dashboard, users, settings pages
└── scripts/
    └── install.sh             # One-command Linux installer
```

## 📥 Installation

### Option A — Docker (recommended)

Requires Docker Engine with the Compose plugin on a **Linux** host (host networking is Linux-only).

```bash
git clone https://github.com/mat-674/naivepanel.git
cd naivepanel
bash scripts/docker-install.sh
```

The script will ask for an optional domain and Let's Encrypt email, build the image (compiles NaiveProxy and the panel from source — takes a few minutes on first run), start the stack, and print credentials plus the SSH tunnel command.

```bash
# Stop (keeps data)
bash scripts/docker-install.sh --down

# Stop and wipe all data
bash scripts/docker-install.sh --uninstall
```

**How it works:**
- Uses `network_mode: host` so Caddy binds the host's public `:443`/`:80`, while the admin panel stays on `127.0.0.1` (loopback only).
- All mutable state (config, SQLite database, Caddyfile, ACME certificates) is stored in the `naivepanel-data` named volume and survives image rebuilds.
- On first boot the entrypoint initialises the database with `--setup`. Subsequent restarts skip init (keyed on the presence of the database file).
- `docker compose pull && docker compose up -d` is the upgrade path; no data is lost.

> **macOS / Windows Docker Desktop:** host networking is not supported. You can still build and develop, but the panel and Caddy won't bind to your host network.

### Option B — Bare-metal (systemd)

Run the following on your Linux server (Ubuntu / Debian / CentOS / Fedora / Arch) **as root**:

```bash
bash <(curl -sL https://raw.githubusercontent.com/mat-674/naivepanel/main/scripts/install.sh)
```

The installer will:
1. Detect your OS and architecture.
2. Install dependencies (Go, xcaddy, git, etc.).
3. Launch the **Setup Wizard** — prompts for domain, TLS email, admin credentials, and first proxy user (all fields optional; auto-generated if left blank).
4. Build NaiveProxy from source (`xcaddy` + `klzgrad/forwardproxy`).
5. Clone and compile NaivePanel from source.
6. Initialize the SQLite database and write the initial Caddyfile.
7. Register and start separate `naiveproxy` and `naivepanel` systemd services.
8. Print the SSH tunnel command, admin credentials, and proxy connection URI.

The admin panel intentionally listens only on `127.0.0.1`. After installation, tunnel it over SSH and open the displayed local URL:

```bash
ssh -L <panel-port>:127.0.0.1:<panel-port> root@<server-ip>
```

#### Update

```bash
bash <(curl -sL https://raw.githubusercontent.com/mat-674/naivepanel/main/scripts/install.sh) --update
```

#### Uninstall

```bash
bash <(curl -sL https://raw.githubusercontent.com/mat-674/naivepanel/main/scripts/install.sh) --uninstall
```

## ⚙️ CLI Flags

The `naivepanel` binary accepts the following flags:

| Flag | Default | Description |
|---|---|---|
| `--config` | auto | Path to `config.json` |
| `--data-dir` | OS default | Data directory (DB, Caddyfile) |
| `--port` | from config | HTTP port for the panel |
| `--bind` | `127.0.0.1` | Loopback address for the administrative panel |
| `--setup` | `false` | Init DB, print credentials, then exit |
| `--domain` | — | NaiveProxy domain |
| `--tls-email` | — | Email for Let's Encrypt |
| `--create-user` | `false` | Create a proxy user during setup |
| `--proxy-user` | auto | Proxy username |
| `--proxy-pass` | auto | Proxy password |
| `--admin-user` | auto | Admin username |
| `--admin-pass` | auto | Admin password |

## 🔌 API Endpoints

All endpoints except `/api/login` require a `Authorization: Bearer <token>` header.

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/login` | Authenticate, receive JWT |
| `GET` | `/api/users` | List all proxy users |
| `POST` | `/api/users` | Create a proxy user |
| `PUT` | `/api/users/{id}` | Update a proxy user |
| `DELETE` | `/api/users/{id}` | Delete a proxy user |
| `GET` | `/api/users/{id}/link` | Get `naive+https://` URI + QR code |
| `GET` | `/api/settings` | Get server settings |
| `PUT` | `/api/settings` | Save server settings |
| `GET` | `/api/status` | Server & NaiveProxy status + traffic totals |
| `POST` | `/api/service/{action}` | Control NaiveProxy: `start` / `stop` / `restart` |

## ⚙️ Post-Install Configuration

1. Create the SSH tunnel shown after installation, then open the local panel URL.
2. **Settings** → enter your domain, TLS email, and optional camouflage (decoy) URL.
3. Click **Apply** — the Caddyfile is regenerated and the `naiveproxy` service is reloaded automatically. Subscription URLs are served at `https://<domain>/<subscription-path>/<token>`.
4. **Users** → add users, set traffic limits, get connection links / QR codes.

## 🔒 Security

- **Loopback Admin Panel**: The panel listens on a randomly assigned loopback port and is reached through SSH tunnelling by default.
- **JWT Auth**: All API endpoints are protected by short-lived JSON Web Tokens.
- **bcrypt**: Admin password stored as a bcrypt hash.
- **Probe Resistance**: NaiveProxy's built-in probe resistance hides the proxy from scanners.
- **Credential Validation**: Proxy usernames and passwords are restricted to URL-safe characters (letters, digits, and `- . _ ~`) so they cannot inject directives into the generated Caddyfile or break the `naive+https://` connection URI. Server settings (domain, TLS email, decoy site, subscription path) are normalized and validated before the Caddyfile is regenerated.

## 📄 License

MIT
