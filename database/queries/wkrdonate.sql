-- name: GetDonables :many

SELECT id, sponsor_id, recipient_id
FROM donations
WHERE
	donate_ts < last_ts AND
	donate_attempt_ts < UNIXEPOCH() - 3600;

-- name: GetAreDonatesFinished :one

WITH
	num_repos_remaining AS (
		SELECT COUNT(rowid) AS num
		FROM repos
		WHERE animate_ts < last_ts
	)
SELECT
	(
		num_repos_remaining.num = 0 AND
		kv.v IS NOT NULL
	) AS is_finished
FROM num_repos_remaining
LEFT JOIN kvstore kv ON k = 'entity-ts';

-- name: UpdateDonationDonateAttemptTs :exec

UPDATE donations
SET donate_attempt_ts = UNIXEPOCH()
WHERE id = ?;

-- name: UpdateDonationDonateTs :exec

UPDATE donations
SET donate_ts = UNIXEPOCH()
WHERE id = ?;

