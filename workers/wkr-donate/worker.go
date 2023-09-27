//go:generate autoquery
package wkrdonate

//
// This worker continuously checks to see if there are any outstanding
// donations and initiates a createSponsorship GH GraphQL call for each.
// An outstanding donation is one which:
// 	- donate_ts is before donable_ts;
//	- donate_ts is before 1st of the current month;
// This results in a monthly donation to the project.
//

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/alecthomas/errors"
	"github.com/shurcooL/githubv4"
	wkrghsponsor "github.com/thnxdev/wkr-gh-sponsor"
	"github.com/thnxdev/wkr-gh-sponsor/database"
	"github.com/thnxdev/wkr-gh-sponsor/utils/log"
	"github.com/thnxdev/wkr-gh-sponsor/workers"
	"golang.org/x/oauth2"
)

func init() {
	workers.Register(New)
}

func New(
	db *database.DB,
	ghAccesstoken wkrghsponsor.GhAccessToken,
	sponsorAmount wkrghsponsor.SponsorAmount,
) workers.Worker {
	return func(ctx context.Context) (bool, error) {
		logger := log.FromContext(ctx)

		client := githubv4.NewClient(
			oauth2.NewClient(
				ctx,
				oauth2.StaticTokenSource(&oauth2.Token{
					AccessToken: string(ghAccesstoken),
				}),
			),
		)

		/* autoquery name: GetDonables :many

		SELECT id, sponsor_id, recipient_id
		FROM donations
		WHERE
			donate_ts < last_ts AND
			donate_attempt_ts < UNIXEPOCH() - 3600;
		*/
		rows, err := db.GetDonables(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// If entities and repos are finished and there are no more rows
				// then this worker should finish as well

				/* autoquery name: GetAreDonatesFinished :one

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
				*/
				isFinished, err := db.GetAreDonatesFinished(ctx)
				if err != nil {
					return false, nil
				}
				if isFinished.(int64) == 1 {
					return false, workers.ErrDone
				}
				return false, nil
			}
			return false, errors.Wrap(err, "failed to get donable rows")
		}

		amount := githubv4.Int(sponsorAmount)
		isRecurring := githubv4.Boolean(true)
		privacyLevel := githubv4.SponsorshipPrivacy(githubv4.SponsorshipPrivacyPublic)

		// For each recipient create a GH sponsorship that is:
		//	- $1
		//	- recurring
		//	- is public
		for _, row := range rows {
			row := row
			logger.Infof("donating %s:%s", row.SponsorID, row.RecipientID)

			var m struct {
				CreateSponsorship struct {
					ClientMutationID string
				} `graphql:"createSponsorship(input:$input)"`
			}
			id := githubv4.String(fmt.Sprintf("%s:%s", row.SponsorID, row.RecipientID))
			sponsorLogin := githubv4.String(row.SponsorID)
			sponsorableLogin := githubv4.String(row.RecipientID)
			var input githubv4.Input = githubv4.CreateSponsorshipInput{
				ClientMutationID: &id,
				IsRecurring:      &isRecurring,
				Amount:           &amount,
				SponsorLogin:     &sponsorLogin,
				SponsorableLogin: &sponsorableLogin,
				PrivacyLevel:     &privacyLevel,
			}

			err := client.Mutate(ctx, &m, input, nil)
			if err != nil {
				logger.WithError(err).Error("failed to create sponsorship")
				/* autoquery name: UpdateDonationDonateAttemptTs :exec

				UPDATE donations
				SET donate_attempt_ts = UNIXEPOCH()
				WHERE id = ?;
				*/
				_ = db.UpdateDonationDonateAttemptTs(ctx, row.ID)
				continue
			}

			/* autoquery name: UpdateDonationDonateTs :exec

			UPDATE donations
			SET donate_ts = UNIXEPOCH()
			WHERE id = ?;
			*/
			_ = db.UpdateDonationDonateTs(ctx, row.ID)
		}

		return false, nil
	}
}
