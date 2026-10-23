# Trackete

**Track anything. Build the dashboard you actually want.**

Trackete is a self-hosted generalist tracker. Create a yes/no habit or a number
with a unit — beers, weight, meditation, pages read — log it in seconds, then
arrange charts on a dashboard that is yours. One container, one SQLite file, no
external services.

![Trackete dashboard in dark mode](docs/screenshots/dashboard-dark.png)

## What you get

- **Yes/No and Number trackers**, each with an optional goal (`at least` / `at
  most`, per day, week or month).
- **One value per day**, quick steppers, and a detail view for editing history.
- **Six chart types** — line, bar, area, donut, calendar heatmap, stat card.
- **Drag-and-resize dashboards**, multiple dashboards, live statistics
  (streaks, completion, strongest weekday, on-target periods).
- **Multi-user accounts** with per-user data isolation, light/dark themes.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the data model, chart
semantics and API.

## Quick start (Docker Compose)

```bash
git clone https://github.com/a-chris/trackete.git
cd trackete
cp .env.example .env          # then set JWT_SECRET in .env
docker compose up -d --build
```

Open <http://localhost:8080> and create an account.

### Prebuilt image (one-liner)

Release tags publish a multi-arch image to GHCR:

```bash
docker run -d --name trackete \
  -p 8080:8080 \
  -v trackete-data:/data \
  -e JWT_SECRET="$(openssl rand -hex 32)" \
  ghcr.io/a-chris/trackete:latest
```

> Until a `v*` tag has been pushed, use the Compose or source build below; the
> release workflow (`.github/workflows/release.yml`) creates the image.

### From source

Requirements: Go 1.27+ and Node 24+. [mise](https://mise.jdx.dev) reads
`.mise.toml` and installs both.

```bash
mise install
make install                  # npm ci
make build                    # frontend/dist + backend/trackete
STATIC_DIR=./frontend/dist DB_PATH=./data/trackete.db ./backend/trackete
```

## Critical configuration

| Variable | Why it matters |
| --- | --- |
| `JWT_SECRET` | **Set this.** Signs session cookies. If unset, a random secret is generated per boot and everyone is logged out on restart. Use `openssl rand -hex 32`. |
| `TRUSTED_PROXIES` | **Set this when behind a reverse proxy** (comma-separated IPs/CIDRs). Empty means no proxy is trusted, so forwarded headers are ignored and `Secure` cookies plus per-IP login throttling would key on the proxy. |
| `ALLOW_REGISTRATION` | Defaults to `true`. Set `false` once your accounts exist. |
| `TOKEN_TTL` | Session lifetime, default `720h` (30 days). |
| `DB_PATH` / volume | Everything lives in the SQLite file (Docker: `/data/trackete.db`). Mount a persistent volume. |
| `PORT` | HTTP port, default `8080`. |
| `STATIC_DIR` | Built SPA directory. Empty = API only (Docker sets `/app/dist`). |
| `CORS_ORIGINS` | Comma-separated extra origins for credentialed requests. Only for cross-origin development; wildcards are ignored. |

## Running behind HTTPS

Terminate TLS at your reverse proxy, forward `X-Forwarded-Proto: https`, and list
the proxy in `TRUSTED_PROXIES`. Cookies then become `Secure` automatically.

Caddy example:

```
trackete.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

With Compose, bind the port to loopback only: `ports: ["127.0.0.1:8080:8080"]`
and put the proxy in front. Set `TRUSTED_PROXIES=127.0.0.1/32` (or the proxy's
Docker network) so forwarded headers are honored.

SQLite runs in WAL mode: to back up, archive the whole volume while the container
is stopped (or use `sqlite3 .backup`).

```bash
docker run --rm -v trackete-data:/data -v "$PWD":/backup alpine \
  tar czf /backup/trackete-backup.tar.gz -C /data .
```

## Development

```bash
make dev        # Go API :8080 + Vite :5173 (proxied, cookies stay same-origin)
make test       # Go vet/tests, TypeScript, ESLint, production build
```

Layout: `backend/` (Go API, SQLite store, embedded migrations, analytics) and
`frontend/` (React, DaisyUI themes, ApexCharts, dashboard grid).

## Notes

- Trackete logs one value per tracker per day; multiple entries per day and
  entry variants/tags are on the [roadmap](docs/ARCHITECTURE.md#roadmap).
- Charts render at most ten years of history; stored rows are never deleted.
- Registration can be closed; duplicate-email responses reveal that an account
  exists (intentional for a self-hosted install — close registration after
  setup).

Security policy: [SECURITY.md](SECURITY.md). Contributions:
[CONTRIBUTING.md](CONTRIBUTING.md). License: [MIT](LICENSE).
