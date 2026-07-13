# syntax=docker/dockerfile:1

# ---- Stage 1: build NaiveProxy (Caddy + forwardproxy) with xcaddy ----
FROM golang:1.23-bookworm AS naive-builder
RUN go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
RUN xcaddy build v2.9.1 \
        --with github.com/caddyserver/forwardproxy=github.com/klzgrad/forwardproxy@master \
        --output /out/naive

# ---- Stage 2: build the NaivePanel binary (CGO for go-sqlite3) ----
FROM golang:1.23-bookworm AS panel-builder
ENV CGO_ENABLED=1
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o /out/naivepanel main.go

# ---- Stage 3: runtime ----
FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Binaries live in the image; mutable state lives in the /opt/naivepanel volume.
COPY --from=naive-builder /out/naive /usr/local/bin/naive
COPY --from=panel-builder /out/naivepanel /usr/local/bin/naivepanel
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/naive /usr/local/bin/naivepanel /usr/local/bin/entrypoint.sh

# Keep Caddy's ACME material inside the persisted volume.
ENV XDG_DATA_HOME=/opt/naivepanel/caddy-data
ENV INSTALL_DIR=/opt/naivepanel

VOLUME /opt/naivepanel
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
