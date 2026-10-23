# Architecture

Deeper notes on how Trackete stores and aggregates data. For setup, see the
[README](../README.md).

## Shape of the system

```
Browser (React SPA)
   │  same-origin fetch, HttpOnly session cookie
   ▼
Go API (Gin) ── SQLite (WAL) inside /data
   │
   └─ serves the built SPA from STATIC_DIR (Docker: /app/dist)
```

One process, one database file, no queue or cache. The API and the SPA ship in
the same image.

## Data model

- **users**: id, email (case-insensitive unique), name, bcrypt hash,
  `token_version` (bumped on password change to invalidate old JWTs).
- **trackers**: name, description, icon, colour, `kind` (`boolean` / `numeric`),
  optional unit, optional goal (JSON `{type, value, period}`), `archived`.
  Goals are `at_least` / `at_most` per `day` / `week` / `month`.
- **entries**: one value per tracker per day.
  `PRIMARY KEY (tracker_id, day)`; `day` is a local calendar date
  (`YYYY-MM-DD`) supplied by the browser, so timezones never shift a log.
  Logging the same day again replaces the value.
- **dashboards**: per-user, at most one `is_default` (enforced by a partial
  unique index). Migration `0003_default_dashboard_backfill.sql` also promotes
  the oldest dashboard for every user who had dashboards but none flagged, so
  existing databases end up with exactly one default.
- **widgets**: dashboard, tracker, chart type, config JSON
  (`range`, `granularity`, `aggregate`, `metric`), 12-column grid position.

Deleting a tracker cascades to its entries and its widgets. Archiving keeps
everything and only hides the tracker from the Today screen and pickers.

## Chart payloads

Every chart request returns the same envelope: `labels`, `series`, optional
`heatmap`/`stat`, `summary`, `empty`. Aggregation is pure Go in
`backend/internal/analytics`, so a chart and the headline numbers cannot
disagree.

- **boolean** trackers: 1/0 per day; weekly/monthly buckets count days on.
- **numeric** trackers: aggregate with **sum** or **average over logged days**.
  Missing days never dilute an average.
- Missing days count as 0 for streaks and per-period goals; a day without a
  record satisfies an `at most` goal.
- **Current streak** is computed from up to ten years of history, independent
  of the visible chart range. **Best streak** is scoped to the selected range.
- **Period goals** score only weeks/months fully contained in the range; a
  clipped boundary period is excluded from the denominator.
- Charts render at most ten years of history (`AllRangeDays`); stored rows are
  never dropped. "All time" widgets start at the tracker's first entry.

## API

All routes live under `/api`. Authentication is the session cookie
(`Authorization: Bearer <token>` also works for API clients). Every query is
scoped by the authenticated user.

| Method | Route | Purpose |
| --- | --- | --- |
| `GET` | `/health`, `/meta` | Health, version, registration flag. |
| `POST` | `/auth/register`, `/auth/login`, `/auth/logout` | Session lifecycle. |
| `GET/PATCH` | `/auth/me` | Current profile. |
| `POST` | `/auth/password` | Change password (invalidates other sessions). |
| `GET/POST` | `/trackers` | List (`?includeArchived=true`) and create. |
| `GET/PUT/DELETE` | `/trackers/:id` | Read, replace, delete. |
| `GET` | `/trackers/:id/data` | Chart payload (`chart, from, to, granularity, aggregate, metric`). |
| `GET/PUT/DELETE` | `/trackers/:id/entries[/:day]` | Read, upsert, delete entries. |
| `GET` | `/overview` | Today screen data (`?today=YYYY-MM-DD`). |
| `GET/POST` | `/dashboards` | List and create dashboards. |
| `GET/PATCH/DELETE` | `/dashboards/:id` | Read, rename, delete. |
| `GET` | `/dashboards/:id/data` | Chart payloads for every widget. |
| `POST` | `/dashboards/:id/widgets` | Add a chart. |
| `PUT/DELETE` | `/widgets/:id` | Update or remove a chart. |
| `PUT` | `/dashboards/:id/layout` | Persist drag/resize positions. |

## Roadmap

The schema and analytics package are built to grow:

- multiple entries per day (doses, meals, sets) with per-entry metadata
- quick variants/tags on entries ("hot beer" vs "cold beer") and distribution charts
- notes surfaced in charts and a searchable journal
- more widgets: stacked bars, scatter, radial, tables, multi-tracker charts
- pattern detection ("you drink most on Saturdays") surfaced as insights
- CSV/JSON import and export
