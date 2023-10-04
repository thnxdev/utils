-- name: GetRepos :one

SELECT owner_name, repo_name, cursor_manifest, cursor_dep
FROM repos
WHERE animate_ts < last_ts
LIMIT 1;

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

