-- name: InsertDonation :exec

INSERT INTO donations (sponsor_id, recipient_id, last_ts)
VALUES (?, ?, ?)
ON CONFLICT (sponsor_id, recipient_id)
DO NOTHING;
