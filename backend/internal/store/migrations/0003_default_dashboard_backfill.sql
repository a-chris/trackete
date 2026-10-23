-- 0003: give every user with dashboards exactly one flagged default.
--
-- 0002 enforced *at most* one default per user with its partial unique index,
-- but it did not create one where every dashboard of a user was left unflagged
-- (a restored or hand-edited database, for instance). Promote the oldest
-- dashboard of each such user. A user with a default is excluded, so the
-- partial unique index can see at most one promotion per user.
UPDATE dashboards
   SET is_default = 1
 WHERE id IN (
   SELECT id FROM (
     SELECT id,
            ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY created_at ASC, id ASC) AS rn
       FROM dashboards
      WHERE user_id NOT IN (SELECT user_id FROM dashboards WHERE is_default = 1)
   ) ranked
    WHERE rn = 1
 );
