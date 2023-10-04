-- name: GetDonables :many

SELECT id, sponsor_id, recipient_id
FROM donations
WHERE
	donate_ts < last_ts AND
	donate_attempt_ts < UNIXEPOCH() - 3600;

-- name: UpdateDonationDonateAttemptTs :exec

UPDATE donations
SET donate_attempt_ts = UNIXEPOCH()
WHERE id = ?;

-- name: UpdateDonationDonateTs :exec

UPDATE donations
SET donate_ts = UNIXEPOCH()
WHERE id = ?;

