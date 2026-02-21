# NaivePanel

**NaivePanel** is a lightweight web management panel for NaiveProxy (Caddy with forwardproxy plugin). It provides a clean dark-themed interface to manage proxy users, monitor traffic, and configure server settings.

## NaivePanel Dashboard
![NaivePanel Dashboard](https://raw.githubusercontent.com/mat-674/naivepanel/main/assets/preview.png)

## 🚀 Features

- **One-Command Installation**: Interactive setup wizard — prompts for domain, TLS email, admin and proxy credentials (all auto-generated if left empty). Builds everything from source.
- **Modern UI**: Dark-themed SPA with glassmorphism aesthetics.
- **Dynamic Caddyfile**: Automatically regenerates and reloads Caddy config on user create/update/delete.
- **Multi-User Support**: Create, update, and delete proxy users with per-user traffic limits and enable/disable toggles.
- **Traffic Statistics**: Tracks per-user upload/download traffic; dashboard shows totals across all users.
- **ACME / Let's Encrypt**: Built-in SSL certificate management via Caddy.
- **Connection Links & QR Codes**: Generates `naive+https://` URIs with QR codes on demand.
- **Service Control**: Start / Stop / Restart NaiveProxy directly from the panel UI.
- **Single Binary**: Written in Go; static frontend assets are embedded via `embed.FS`.

## 🛠 Tech Stack

| Layer | Technology |
|---|---|
| **Backend** | Go 1.21+, `net/http`, SQLite (`go-sqlite3`), JWT (`golang-jwt/jwt v5`), bcrypt (`golang.org/x/crypto`) |
| **Frontend** | Vanilla HTML / CSS / JS — SPA, no build step required |
| **QR Codes** | `skip2/go-qrcode` — base64-encoded PNG served via API |
| **Proxy Core** | NaiveProxy (Caddy v2.9.1 + `klzgrad/forwardproxy`) |

## � Project Structure

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
7. Register and start a systemd service.
8. Print the panel URL, admin credentials, and proxy connection URI.

### Uninstall

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
| `POST` | `/api/settings` | Save server settings |
| `GET` | `/api/status` | Server & NaiveProxy status + traffic totals |
| `POST` | `/api/service/{action}` | Control NaiveProxy: `start` / `stop` / `restart` |

## ⚙️ Post-Install Configuration

1. Open the panel at the URL shown after installation.
2. **Settings** → enter your domain, TLS email, and optional camouflage (decoy) URL.
3. Click **Apply** — the Caddyfile is regenerated and Caddy is reloaded automatically.
4. **Users** → add users, set traffic limits, get connection links / QR codes.

## 🔒 Security

- **Random Port**: The panel listens on a randomly assigned port (printed at install).
- **JWT Auth**: All API endpoints are protected by short-lived JSON Web Tokens.
- **bcrypt**: Admin password stored as a bcrypt hash.
- **Probe Resistance**: NaiveProxy's built-in probe resistance hides the proxy from scanners.

## 📄 License

MIT
