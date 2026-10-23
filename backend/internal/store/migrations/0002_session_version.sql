-- 0002: session versioning plus the one-default-dashboard invariant.
--
-- token_version is embedded in the session JWT as the "ver" claim. Bumping it
-- (on password change) invalidates every session issued before the change.
ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;

-- Exactly one default dashboard per user. Existing databases may hold several
-- rows flagged as default, so demote everything but the oldest row first, then
-- enforce the invariant with a partial unique index.
UPDATE dashboards
   SET is_default = 0
 WHERE is_default = 1
   AND id NOT IN (
     SELECT id FROM (
       SELECT id,
              ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at ASC, id ASC) AS rn
         FROM dashboards
        WHERE is_default = 1
     ) ranked
      WHERE rn = 1
   );

CREATE UNIQUE INDEX idx_dashboards_one_default
    ON dashboards (user_id)
 WHERE is_default = 1;
