//go:generate autoquery
package tddonate

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
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"
	wkrghsponsor "github.com/thnxdev/wkr-gh-sponsor"
	"github.com/thnxdev/wkr-gh-sponsor/database"
	"github.com/thnxdev/wkr-gh-sponsor/utils/log"
	"github.com/thnxdev/wkr-gh-sponsor/workers"
	"golang.org/x/oauth2"
)

func New(
	db *database.DB,
	ghAccesstoken wkrghsponsor.GhAccessToken,
	sponsorAmount wkrghsponsor.SponsorAmount,
) workers.Worker {
	return func(ctx context.Context) error {
		logger := log.FromContext(ctx)

		client := githubv4.NewClient(
			oauth2.NewClient(
				ctx,
				oauth2.StaticTokenSource(&oauth2.Token{
					AccessToken: string(ghAccesstoken),
				}),
			),
		)

		now := time.Now()
		som := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

		/* autoquery name: GetDonables :many

		SELECT id, sponsor_id, recipient_id
		FROM donations
		WHERE
			donate_ts < donable_ts AND
			donate_attempt_ts < UNIXEPOCH() - 3600 AND
			donate_ts < ?;
		*/
		rows, err := db.GetDonables(ctx, som.Unix())
		if err != nil {
			logger.WithError(err).Error("failed to get donable rows")
			return err
		}

		amount := githubv4.Int(sponsorAmount)
		isRecurring := githubv4.Boolean(false)
		privacyLevel := githubv4.SponsorshipPrivacy(githubv4.SponsorshipPrivacyPublic)

		// For each recipient create a GH sponsorship that is:
		//	- $1
		//	- not recurring
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

		return nil
	}
}
