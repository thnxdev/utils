-- name: KvstoreGetLastStatus :one

WITH d AS (VALUES(1))
SELECT
	kvc.v AS next_page,
	kvt.v AS entity_ts
FROM d
LEFT JOIN kvstore kvc ON kvc.k = 'next-page'
LEFT JOIN kvstore kvt ON kvt.k = 'entity-ts';

-- name: ReposInsert :exec

INSERT INTO repos (owner_name, repo_name, last_ts)
VALUES (?, ?, UNIXEPOCH())
ON CONFLICT (owner_name, repo_name)
DO NOTHING;

-- name: KvstoreInsertNextPage :exec

INSERT INTO kvstore (k, v)
VALUES ('next-page', ?)
ON CONFLICT (k)
DO UPDATE
	SET v = EXCLUDED.v;

-- name: KvstoreInsertEntityTs :exec

INSERT INTO kvstore (k, v)
VALUES ('entity-ts', ?)
ON CONFLICT (k)
DO UPDATE
	SET v = EXCLUDED.v;

