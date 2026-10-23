# Contributing

Thanks for taking a look. Trackete is small on purpose: a Go API, a React app,
and SQLite in a single container.

## Development setup

Requirements: Go 1.27+, Bun 1.4+. [mise](https://mise.jdx.dev) reads `.mise.toml`
and installs both.

```bash
mise install
make install        # bun install
make dev            # API on :8080 (Air live reload), Vite on :5173
```

The first `make dev` installs a pinned Air into `backend/.bin/` (gitignored).
Editing Go files or embedded SQL migrations restarts the API automatically;
Vite hot-reloads the React app.

The Vite dev server proxies `/api` to `http://localhost:8080`, so cookies stay
same-origin in development.

## Before opening a pull request

```bash
make test           # go vet + go test, tsc, eslint, vite build
```

Keep changes focused. If you add a tracker type, a chart type, or an endpoint,
update the README's feature and API sections in the same PR.

## Database changes

Migrations live in `backend/internal/store/migrations` and are embedded into the
binary. Add a new numbered file (`0002_...sql`) — never edit an applied
migration. Each migration runs in a transaction and is recorded in
`schema_migrations`.

## Commit messages

Short subject, body only when it explains *why*. No co-author trailers.
