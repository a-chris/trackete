# Security

## Reporting a vulnerability

Please do not open a public issue for security problems. Use GitHub's private
vulnerability reporting (Security → Advisories → Report a vulnerability) or
email the maintainer listed on the repository profile.

Include the affected version, a reproduction, and the impact you expect. You
will get an acknowledgement as soon as possible, and credit in the release
notes if you want it.

## Deployment checklist

Trackete is designed for self-hosting. Before exposing it to the internet:

- Set `JWT_SECRET` to a long random value (`openssl rand -hex 32`). Without it,
  sessions are invalidated on every restart.
- Terminate TLS at a reverse proxy and forward `X-Forwarded-Proto: https` so
  session cookies are marked `Secure`.
- Set `ALLOW_REGISTRATION=false` once your accounts exist.
- If a reverse proxy sits in front of the container, set `TRUSTED_PROXIES` to its
  IPs/CIDRs. Left empty, `X-Forwarded-For` is ignored (good: it cannot be spoofed),
  but every request then appears to come from the proxy and per-IP rate limiting
  becomes per-proxy.
- Only list explicit origins in `CORS_ORIGINS`; `*` is ignored because cookies
  are sent with credentials.
- Back up the `/data` volume (SQLite database plus WAL files).
- Keep the container image up to date.

Registration answers `409 Conflict` for an email that is already registered. That
is intentional: registration is expected to be closed (`ALLOW_REGISTRATION=false`)
once the intended accounts exist, and the conflict is what a returning user sees
during setup. It is a documented decision, not an accidental account oracle.

## What Trackete does to protect itself

- Passwords are hashed with bcrypt (cost 12); inputs above bcrypt's 72-byte
  limit are rejected with a 400 instead of failing the hash.
- Sessions are signed JWTs delivered in `HttpOnly`, `SameSite=Lax` cookies.
  Each token carries a session version matching `users.token_version`, so a
  password change invalidates every session issued before it.
- Login and registration attempts are rate limited per IP, with stale limiter
  entries evicted periodically so the map cannot grow without bound.
- Every query is scoped by the authenticated user ID; there are no shared rows.
- Responses carry a strict Content-Security-Policy and other hardening headers.
