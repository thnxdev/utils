//go:generate autoquery
package donate

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
	utils "github.com/thnxdev/utils"
	"github.com/thnxdev/utils/database"
	"github.com/thnxdev/utils/utils/log"
	"golang.org/x/oauth2"
)

type CmdDonate struct {
	GhClassicAccessToken utils.GhAccessToken `help:"GitHub classis access token with admin:org & user scopes." required:"" env:"GH_CLASSIC_ACCESS_TOKEN"`
	Amount               int                 `help:"The amount to donate to each dependency." default:"1"`
	IsRecurring          bool                `help:"Whether the donation should be recurring monthly." default:"true"`
}

func (c *CmdDonate) Run(
	ctx context.Context,
	db *database.DB,
) error {
	logger := log.FromContext(ctx)
	logger.Info("starting")

	client := githubv4.NewClient(
		oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(&oauth2.Token{
				AccessToken: string(c.GhClassicAccessToken),
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
			return nil
		}
		return errors.Wrap(err, "failed to get donable rows")
	}

	amount := githubv4.Int(c.Amount)
	isRecurring := githubv4.Boolean(c.IsRecurring)
	privacyLevel := githubv4.SponsorshipPrivacy(githubv4.SponsorshipPrivacyPublic)
	receiveEmails := githubv4.Boolean(false)

	sponsorIds := map[string]string{}

	// For each recipient create a GH sponsorship that is:
	//	- $1
	//	- recurring
	//	- is public
	for _, row := range rows {
		row := row
		logger.Infof("donating %s:%s", row.SponsorID, row.RecipientID)

		failed := false

		sid, ok := sponsorIds[row.SponsorID]
		if !ok {
			var q struct {
				RepositoryOwner struct {
					ID string
				} `graphql:"repositoryOwner(login: $login)"`
			}
			var vars map[string]any = map[string]any{
				"login": githubv4.String(row.SponsorID),
			}

			err := client.Query(ctx, &q, vars)
			if err != nil {
				logger.WithError(err).Error("failed to get sponsor id")
				failed = true
			} else {
				sid = q.RepositoryOwner.ID
				sponsorIds[row.SponsorID] = sid
			}
		}

		if !failed {
			var m struct {
				CreateSponsorship struct {
					ClientMutationID string
				} `graphql:"createSponsorship(input:$input)"`
			}
			id := githubv4.String(fmt.Sprintf("%s:%s", row.SponsorID, row.RecipientID))
			sponsorId := githubv4.ID(sid)
			sponsorableLogin := githubv4.String(row.RecipientID)
			var input githubv4.Input = githubv4.CreateSponsorshipInput{
				ClientMutationID: &id,
				IsRecurring:      &isRecurring,
				Amount:           &amount,
				SponsorID:        &sponsorId,
				SponsorableLogin: &sponsorableLogin,
				PrivacyLevel:     &privacyLevel,
				ReceiveEmails:    &receiveEmails,
			}

			err := client.Mutate(ctx, &m, input, nil)
			if err != nil {
				logger.WithError(err).Errorf("failed to create sponsorship for %s", row.RecipientID)
				failed = true
			}
		}

		if failed {
			/* autoquery name: UpdateDonationDonateAttemptTs :exec

			UPDATE donations
			SET donate_attempt_ts = UNIXEPOCH()
			WHERE id = ?;
			*/
			_ = db.UpdateDonationDonateAttemptTs(ctx, row.ID)
		} else {
			/* autoquery name: UpdateDonationDonateTs :exec

			UPDATE donations
			SET donate_ts = UNIXEPOCH()
			WHERE id = ?;
			*/
			_ = db.UpdateDonationDonateTs(ctx, row.ID)
		}
	}

	return nil
}
