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
- **Shared trackers**: invite people by email, each member logs their own value
  per day, and the charts show one series per member with a combined summary.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the data model, chart
semantics and API.

## Quick start (Docker Compose)

```bash
git clone https://github.com/a-chris/trackete.git
cd trackete
cp .env.example .env          # optional: edit it to pin JWT_SECRET, ports, etc.
docker compose up -d --build
```

Open <http://localhost:8080> and create an account.

### Prebuilt image (one-liner)

Release tags publish a multi-arch image to GHCR:

```bash
docker run -d --name trackete \
  -p 8080:8080 \
  -v trackete-data:/data \
  ghcr.io/a-chris/trackete:latest
```

The signing secret is generated into `/data/.jwt-secret` on first start, so a
container that owns its volume needs no configuration at all. Pass
`-e JWT_SECRET=...` if you would rather manage it yourself.

> Until a `v*` tag has been pushed, use the Compose or source build below; the
> release workflow (`.github/workflows/release.yml`) creates the image.

### From source

Requirements: Go 1.27+ and Bun 1.4+. [mise](https://mise.jdx.dev) reads
`.mise.toml` and installs both.

```bash
mise install
make install                  # bun install
make build                    # frontend/dist + backend/trackete
STATIC_DIR=./frontend/dist DB_PATH=./data/trackete.db ./backend/trackete
```

## Critical configuration

| Variable | Why it matters |
| --- | --- |
| `JWT_SECRET` | Signs session cookies. Optional: if unset, a secret is generated on first run and stored in `.jwt-secret` next to the database (inside the volume), so sessions survive restarts. Set it explicitly to pin the value, or to rotate (set a new one and delete the file). If the secret cannot be stored, startup fails instead of quietly signing everyone out on the next restart. Use `openssl rand -hex 32`. |
| `JWT_SECRET_FILE` | Only used when `JWT_SECRET` is empty: path of the file that holds the generated secret. Defaults to `.jwt-secret` in the database directory. |
| `TRUSTED_PROXIES` | **Set this when behind a reverse proxy** (comma-separated IPs/CIDRs). Empty means no proxy is trusted: the forwarded client IP is ignored (so per-IP login throttling keys on the proxy) and `X-Forwarded-Proto` only marks cookies `Secure` when it comes from a trusted proxy. |
| `ALLOW_REGISTRATION` | Defaults to `true`. Set `false` once your accounts exist. |
| `TOKEN_TTL` | Session lifetime, default `720h` (30 days). |
| `DB_PATH` / volume | Everything lives in the SQLite file (Docker: `/data/trackete.db`). Mount a persistent volume. When overriding `DB_PATH`, keep it under `/data`, or the database lands outside the volume and is lost on restart. |
| `PORT` | HTTP port, default `8080`. |
| `STATIC_DIR` | Built SPA directory. Empty = API only (Docker sets `/app/dist`). |
| `CORS_ORIGINS` | Comma-separated extra origins for credentialed requests. Only for cross-origin development; wildcards are ignored. |
| `APP_URL` | Externally reachable origin used to build invitation links, default `http://localhost:8080`. Set it to the address people actually open (for example `https://trackete.example.com`), otherwise invite emails point at the wrong host. |
| `MAIL_TRANSPORT` | How invitation emails leave the process: `smtp`, `file` or `log`. Empty means `smtp` when `SMTP_HOST` is set, otherwise `file`. |
| `SMTP_HOST`, `SMTP_PORT` | Mail server and port (default `587`, STARTTLS is used when offered). Without a host, invitations stay `pending` and can be resent later. |
| `SMTP_USERNAME`, `SMTP_PASSWORD` | Credentials for SMTP AUTH, if your server needs them. |
| `MAIL_FROM` | Sender address, default `Trackete <no-reply@trackete.local>`. Many providers require a verified sender. |
| `MAIL_FILE_PATH` | Outbox file for the `file` transport, default `outbox.log` next to `DB_PATH` (so `/data/outbox.log` in Docker). This is where invite links appear when no SMTP server is configured. |

## Sharing trackers

A tracker owner can invite people by email, from the tracker form or the Share
button on a tracker that already has members. Invitees receive a link that stays
valid for seven days:

- An existing account is signed in by accepting the link.
- A new invitee picks a display name and password on the acceptance page. This
  works even with `ALLOW_REGISTRATION=false`, so an invite-only instance is
  possible: close registration and hand out invitations.
- Members can log their own value per day and see every member's entries and
  charts. Only the owner can edit the tracker, invite people or remove members;
  a member can change only their own entries.

Invitations are always recorded, even when delivery fails — such an invitation
shows as *not delivered* and the owner can resend it. Invitation history survives
rejection and removal so the invite field can suggest addresses you used before.

Without an SMTP server the `file` transport writes each message, links included,
to `MAIL_FILE_PATH` (`outbox.log` next to the database by default):

```bash
docker compose exec trackete tail -n 40 /data/outbox.log
```

Copy the `/invite/...` link from that file to accept the invitation yourself
during development. `MAIL_TRANSPORT=log` writes the same content to the container
log instead.

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
is stopped (or use `sqlite3 .backup`). Compose pins the volume name to
`trackete-data` (see `docker-compose.yml`), so these commands target exactly the
volume the container uses, whether it was created by Compose or by `docker run`.

```bash
docker compose stop trackete
docker run --rm -v trackete-data:/data -v "$PWD":/backup alpine \
  tar czf /backup/trackete-backup.tar.gz -C /data .
docker compose start trackete
```

## Development

```bash
make dev        # API with live reload (Air) on :8080 + Vite on :5173
make test       # Go vet/tests, TypeScript, ESLint, production build
```

`make dev` runs the API and the Vite dev server side by side; Vite proxies
`/api` to the API, so cookies stay same-origin. Editing a Go file (or an
embedded SQL migration) rebuilds and restarts the API — the first run installs
a pinned Air into `backend/.bin/`. The dev database lives in `data/`
(`DB_PATH`, gitignored).

Layout: `backend/` (Go API, SQLite store, embedded migrations, analytics) and
`frontend/` (React, DaisyUI themes, ApexCharts, dashboard grid).

## Notes

- Each member logs one value per tracker per day; multiple entries per member per
  day and entry variants/tags are on the [roadmap](docs/ARCHITECTURE.md#roadmap).
- Charts render at most ten years of history; stored rows are never deleted.
- Registration can be closed; duplicate-email responses reveal that an account
  exists (intentional for a self-hosted install — close registration after
  setup).

Security policy: [SECURITY.md](SECURITY.md). Contributions:
[CONTRIBUTING.md](CONTRIBUTING.md). License: [MIT](LICENSE).
