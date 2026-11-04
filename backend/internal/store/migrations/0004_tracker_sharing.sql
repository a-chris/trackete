CREATE TABLE tracker_members (
    tracker_id TEXT NOT NULL REFERENCES trackers(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('owner', 'member')),
    created_at TEXT NOT NULL,
    PRIMARY KEY (tracker_id, user_id)
);

CREATE INDEX idx_tracker_members_user ON tracker_members (user_id);

CREATE TABLE invitations (
    id         TEXT PRIMARY KEY,
    tracker_id TEXT NOT NULL REFERENCES trackers(id) ON DELETE CASCADE,
    invited_by TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email      TEXT NOT NULL COLLATE NOCASE,
    status     TEXT NOT NULL CHECK (status IN ('pending', 'delivered', 'accepted', 'rejected', 'removed')),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    settled_at TEXT
);

CREATE INDEX idx_invitations_tracker ON invitations (tracker_id, email);
CREATE INDEX idx_invitations_inviter ON invitations (invited_by);

-- One live invitation per address and tracker. Settled rows are history and
-- stay around so the owner's suggestions keep offering removed invitees.
CREATE UNIQUE INDEX idx_invitations_live ON invitations (tracker_id, email)
    WHERE status IN ('pending', 'delivered');

-- Every existing tracker gets its creator as the accepted owner. Without this
-- row the membership-scoped reads below would hide every tracker that predates
-- sharing.
INSERT INTO tracker_members (tracker_id, user_id, role, created_at)
    SELECT id, user_id, 'owner', created_at FROM trackers;

-- Entries become per member: two people may log the same day on one tracker.
CREATE TABLE entries_scoped (
    tracker_id TEXT NOT NULL REFERENCES trackers(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day        TEXT NOT NULL,
    value      REAL NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tracker_id, user_id, day)
);

INSERT INTO entries_scoped (tracker_id, user_id, day, value, note, created_at, updated_at)
    SELECT tracker_id, user_id, day, value, note, created_at, updated_at FROM entries;

DROP TABLE entries;

ALTER TABLE entries_scoped RENAME TO entries;

CREATE INDEX idx_entries_user_day ON entries (user_id, day);
CREATE INDEX idx_entries_tracker_day ON entries (tracker_id, day);
