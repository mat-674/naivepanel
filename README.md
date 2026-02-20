# NaivePanel

**NaivePanel** is a powerful, lightweight, and beautiful web management panel for NaiveProxy (Caddy with forwardproxy plugin). Inspired by the simplicity and functionality of 3x-ui, it provides a seamless interface to manage your proxy users, monitor traffic, and configure server settings with zero hassle.

## NaivePanel Dashboard
## ![NaivePanel Dashboard](https://raw.githubusercontent.com/mat-674/naivepanel/main/assets/preview.png)

## 🚀 Features

- **One-Command Installation**: Deploy a fully functional NaiveProxy server and panel in seconds (builds from source for maximum compatibility).
- **Modern UI**: A premium dark-themed interface with glassmorphism aesthetics and responsive design.
- **Dynamic Caddyfile**: Automatically generates and reloads Caddy configuration on the fly.
- **Multi-User Support**: Create and manage multiple users with individual traffic limits.
- **Traffic Monitoring**: Real-time server status, uptime, and traffic statistics.
- **ACME Support**: Built-in Let's Encrypt SSL certificate management via Caddy.
- **Connection Links**: Instantly generate `naive+https://` URI and QR codes for clients.
- **Zero-Dependency Core**: Written in Go, static assets are embedded into a single binary.

## 🛠 Tech Stack

- **Backend**: Go (net/http, SQLite, JWT)
- **Frontend**: Vanilla HTML/CSS/JS (SPA architecture)
- **Proxy Core**: NaiveProxy (Caddy + forwardproxy)

## 📥 Installation

Run the following command on your Linux server (Ubuntu/Debian/CentOS/Fedora) as root:

```bash
bash <(curl -sL https://raw.githubusercontent.com/mat-674/naivepanel/main/scripts/install.sh)
```

### Options

**Uninstall NaivePanel:**
```bash
bash <(curl -sL https://raw.githubusercontent.com/mat-674/naivepanel/main/scripts/install.sh) --uninstall
```

## ⚙️ Configuration

1. **Access the Panel**: Use the URL and credentials printed at the end of the installation.
2. **Domain Setup**: Go to the **Settings** page and enter your domain name.
3. **TLS/SSL**: Provide an email address for Let's Encrypt to enable automatic SSL certificate issuance.
4. **Decoy Site**: Set a "Camouflage" URL (e.g., `https://www.wikipedia.org`) to show when someone probes your server.
5. **Add Users**: Go to the **Users** tab, add an account, and copy the connection link.

## 🔒 Security

- **Random Port**: The panel runs on a randomly generated port by default.
- **JWT Auth**: All API requests are protected by JSON Web Tokens.
- **Privacy**: High-performance "probe resistance" provided by NaiveProxy's design.

## 📄 License

MIT
