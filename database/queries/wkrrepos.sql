-- name: GetRepos :one

SELECT owner_name, repo_name, cursor_manifest, cursor_dep
FROM repos
WHERE animate_ts < last_ts
LIMIT 1;

-- name: GetAreReposFinished :one

WITH d AS (VALUES(1))
SELECT kv.v IS NOT NULL AS is_finished
FROM d
LEFT JOIN kvstore kv ON k = 'entity-ts';

-- name: InsertDonation :exec

INSERT INTO donations (sponsor_id, recipient_id, last_ts)
VALUES (?, ?, ?)
ON CONFLICT (sponsor_id, recipient_id)
DO NOTHING;

-- name: RepoUpdateCursorDep :exec

UPDATE repos
SET cursor_dep = ?
WHERE owner_name = ? AND repo_name = ?;

-- name: RepoUpdateCursorManifest :exec

UPDATE repos
SET cursor_manifest = ?
WHERE owner_name = ? AND repo_name = ?;

-- name: RepoUpdateAnimateTs :exec

UPDATE repos
SET animate_ts = UNIXEPOCH()
WHERE owner_name = ? AND repo_name = ?;

