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
- Back up the `/data` volume (SQLite database plus WAL files).
- Keep the container image up to date.

## What Trackete does to protect itself

- Passwords are hashed with bcrypt (cost 12).
- Sessions are signed JWTs delivered in `HttpOnly`, `SameSite=Lax` cookies.
- Login and registration attempts are rate limited per IP.
- Every query is scoped by the authenticated user ID; there are no shared rows.
- Responses carry a strict Content-Security-Policy and other hardening headers.
