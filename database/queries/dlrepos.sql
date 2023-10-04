-- name: ReposInsert :exec

INSERT INTO repos (owner_name, repo_name, last_ts)
VALUES (?, ?, UNIXEPOCH())
ON CONFLICT (owner_name, repo_name)
DO NOTHING;

