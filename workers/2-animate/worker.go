//go:generate autoquery
package wkranimate

//
// This worker checks all the latest donations and for each recipient
// does a GH GraphQL call to check if they are set up for GH Sponsors.
//

import (
	"context"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	wkrghsponsor "github.com/thnxdev/wkr-gh-sponsor"
	"github.com/thnxdev/wkr-gh-sponsor/database"
	"github.com/thnxdev/wkr-gh-sponsor/utils/log"
	"github.com/thnxdev/wkr-gh-sponsor/workers"
)

func New(
	db *database.DB,
	ghAccesstoken wkrghsponsor.GhAccessToken,
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

		/* autoquery name: GetAnimatables :many

		SELECT id, sponsor_id, recipient_id
		FROM donations
		WHERE
			animate_ts < last_ts AND
			animate_attempt_ts < UNIXEPOCH() - 3600;
		*/
		rows, err := db.GetAnimatables(ctx)
		if err != nil {
			logger.WithError(err).Error("failed to get animatable rows")
			return err
		}

		for _, row := range rows {
			logger.Infof("animating %s:%s", row.SponsorID, row.RecipientID)

			type (
				Sponsorable struct {
					HasSponsorsListing bool
				}
			)
			var q struct {
				RepositoryOwner struct {
					Sponsorable `graphql:"... on Sponsorable"`
				} `graphql:"repositoryOwner(login: $login)"`
			}
			var vars map[string]any = map[string]any{
				"login": githubv4.String(row.RecipientID),
			}

			err := client.Query(ctx, &q, vars)
			if err != nil {
				logger.WithError(err).Error("failed to query recipient account")
				/* autoquery name: UpdateDonationAnimateAttemptTs :exec

				UPDATE donations
				SET animate_attempt_ts = UNIXEPOCH()
				WHERE id = ?;
				*/
				_ = db.UpdateDonationAnimateAttemptTs(ctx, row.ID)
				continue
			}

			if q.RepositoryOwner.HasSponsorsListing {
				/* autoquery name: UpdateDonationDonableAnimateTs :exec

				UPDATE donations
				SET
					animate_ts = UNIXEPOCH(),
					donable_ts = UNIXEPOCH()
				WHERE id = ?;
				*/
				_ = db.UpdateDonationDonableAnimateTs(ctx, row.ID)
			} else {
				/* autoquery name: UpdateDonationAnimateTs :exec

				UPDATE donations
				SET animate_ts = UNIXEPOCH()
				WHERE id = ?;
				*/
				_ = db.UpdateDonationAnimateTs(ctx, row.ID)
			}
		}

		return nil
	}
}
