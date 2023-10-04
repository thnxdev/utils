//go:generate autoquery
package dlrepos

import (
	"context"

	"github.com/alecthomas/errors"
	"github.com/google/go-github/v55/github"
	utils "github.com/thnxdev/utils"
	"github.com/thnxdev/utils/database"
	"github.com/thnxdev/utils/utils/log"
)

type CmdDlRepos struct {
	GhClassicAccessToken utils.GhAccessToken `help:"GitHub classis access token with admin:org & user scopes." required:"" env:"GH_CLASSIC_ACCESS_TOKEN"`
	Entities             []utils.Entity      `help:"The GitHub entities to import for sponsorships." required:""`
}

func (c *CmdDlRepos) Run(
	ctx context.Context,
	db *database.DB,
) error {
	logger := log.FromContext(ctx)
	logger.Info("starting")

	nextPage := 0

	for {
		client := github.NewClient(nil).WithAuthToken(string(c.GhClassicAccessToken))
		repos, resp, err := client.Repositories.List(ctx, "", &github.RepositoryListOptions{
			ListOptions: github.ListOptions{
				PerPage: 100,
				Page:    nextPage,
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to get repositories")
		}

		for _, r := range repos {
			validEntity := false
			for _, e := range c.Entities {
				if *r.Owner.Login == string(e) {
					validEntity = true
					break
				}
			}

			if !validEntity {
				logger.Infof("%s ignored", *r.FullName)
				continue
			}

			logger.Infof("%s added", *r.FullName)

			/* autoquery name: ReposInsert :exec

			INSERT INTO repos (owner_name, repo_name, last_ts)
			VALUES (?, ?, UNIXEPOCH())
			ON CONFLICT (owner_name, repo_name)
			DO NOTHING;
			*/
			err := db.ReposInsert(ctx, database.ReposInsertParams{
				OwnerName: *r.Owner.Login,
				RepoName:  *r.Name,
			})
			if err != nil {
				return errors.Wrap(err, "failed to insert repos")
			}
		}

		if resp.NextPage == 0 {
			break
		}
		nextPage = resp.NextPage
	}

	return nil
}
