-- name: GetAnimatables :many

SELECT id, sponsor_id, recipient_id
FROM donations
WHERE
	animate_ts < last_ts AND
	animate_attempt_ts < UNIXEPOCH() - 3600;

-- name: UpdateDonationAnimateAttemptTs :exec

UPDATE donations
SET animate_attempt_ts = UNIXEPOCH()
WHERE id = ?;

-- name: UpdateDonationDonableAnimateTs :exec

UPDATE donations
SET
	animate_ts = UNIXEPOCH(),
	donable_ts = UNIXEPOCH()
WHERE id = ?;

-- name: UpdateDonationAnimateTs :exec

UPDATE donations
SET animate_ts = UNIXEPOCH()
WHERE id = ?;

