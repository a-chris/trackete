CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL COLLATE NOCASE UNIQUE,
    name          TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

CREATE TABLE trackers (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon        TEXT NOT NULL DEFAULT '',
    color       TEXT NOT NULL DEFAULT '',
    kind        TEXT NOT NULL CHECK (kind IN ('boolean', 'numeric')),
    unit        TEXT NOT NULL DEFAULT '',
    goal        TEXT,
    archived    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

CREATE INDEX idx_trackers_user ON trackers (user_id, archived);

CREATE TABLE entries (
    tracker_id TEXT NOT NULL REFERENCES trackers(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day        TEXT NOT NULL,
    value      REAL NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tracker_id, day)
);

CREATE INDEX idx_entries_user_day ON entries (user_id, day);

CREATE TABLE dashboards (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    is_default INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_dashboards_user ON dashboards (user_id);

CREATE TABLE widgets (
    id           TEXT PRIMARY KEY,
    dashboard_id TEXT NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
    tracker_id   TEXT REFERENCES trackers(id) ON DELETE CASCADE,
    title        TEXT NOT NULL DEFAULT '',
    chart_type   TEXT NOT NULL CHECK (chart_type IN ('line', 'bar', 'area', 'donut', 'heatmap', 'stat')),
    config       TEXT NOT NULL DEFAULT '{}',
    pos_x        INTEGER NOT NULL DEFAULT 0,
    pos_y        INTEGER NOT NULL DEFAULT 0,
    width        INTEGER NOT NULL DEFAULT 4,
    height       INTEGER NOT NULL DEFAULT 3,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);

CREATE INDEX idx_widgets_dashboard ON widgets (dashboard_id);
