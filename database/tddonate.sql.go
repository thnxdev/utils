// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.21.0
// source: tddonate.sql

package database

import (
	"context"
)

const getDonables = `-- name: GetDonables :many

SELECT id, sponsor_id, recipient_id
FROM donations
WHERE
	donate_ts < donable_ts AND
	donate_attempt_ts < UNIXEPOCH() - 3600 AND
	donate_ts < ?
`

type GetDonablesRow struct {
	ID          int64
	SponsorID   string
	RecipientID string
}

func (q *Queries) GetDonables(ctx context.Context, donateTs int64) ([]GetDonablesRow, error) {
	rows, err := q.db.QueryContext(ctx, getDonables, donateTs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetDonablesRow
	for rows.Next() {
		var i GetDonablesRow
		if err := rows.Scan(&i.ID, &i.SponsorID, &i.RecipientID); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateDonationDonateAttemptTs = `-- name: UpdateDonationDonateAttemptTs :exec

UPDATE donations
SET donate_attempt_ts = UNIXEPOCH()
WHERE id = ?
`

func (q *Queries) UpdateDonationDonateAttemptTs(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, updateDonationDonateAttemptTs, id)
	return err
}

const updateDonationDonateTs = `-- name: UpdateDonationDonateTs :exec

UPDATE donations
SET donate_ts = UNIXEPOCH()
WHERE id = ?
`

func (q *Queries) UpdateDonationDonateTs(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, updateDonationDonateTs, id)
	return err
}
