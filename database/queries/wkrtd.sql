-- name: InsertDonation :exec

INSERT INTO donations (sponsor_id, recipient_id, last_ts)
VALUES (?, ?, UNIXEPOCH())
ON CONFLICT (sponsor_id, recipient_id)
DO UPDATE
	SET last_ts = EXCLUDED.last_ts;

