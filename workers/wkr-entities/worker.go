//go:generate autoquery
package wkrentities

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/alecthomas/errors"
	"github.com/google/go-github/v55/github"
	utils "github.com/thnxdev/utils"
	"github.com/thnxdev/utils/database"
	"github.com/thnxdev/utils/workers"
)

func init() {
	workers.Register(New)
}

func New(
	db *database.DB,
	ghAccesstoken utils.GhAccessToken,
	donorEntities []utils.Entity,
) workers.Worker {
	return func(ctx context.Context) (bool, error) {
		/* autoquery name: KvstoreGetLastStatus :one

		WITH d AS (VALUES(1))
		SELECT
			kvc.v AS next_page,
			kvt.v AS entity_ts
		FROM d
		LEFT JOIN kvstore kvc ON kvc.k = 'next-page'
		LEFT JOIN kvstore kvt ON kvt.k = 'entity-ts';
		*/
		v, err := db.KvstoreGetLastStatus(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, errors.Wrap(err, "failed to get last status")
		}

		if v.EntityTs.Valid {
			return false, workers.ErrDone
		}

		nextPage := 0
		if v.NextPage.Valid {
			nextPage, _ = strconv.Atoi(v.NextPage.String)
		}

		client := github.NewClient(nil).WithAuthToken(string(ghAccesstoken))
		repos, resp, err := client.Repositories.List(ctx, "", &github.RepositoryListOptions{
			ListOptions: github.ListOptions{
				PerPage: 100,
				Page:    nextPage,
			},
		})
		if err != nil {
			return false, errors.Wrap(err, "failed to get repositories")
		}

		for _, r := range repos {
			validEntity := false
			for _, e := range donorEntities {
				if *r.Owner.Login == string(e) {
					validEntity = true
					break
				}
			}
			if !validEntity {
				continue
			}

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
				return false, errors.Wrap(err, "failed to insert repos")
			}
		}

		if resp.NextPage != 0 {
			/* autoquery name: KvstoreInsertNextPage :exec

			INSERT INTO kvstore (k, v)
			VALUES ('next-page', ?)
			ON CONFLICT (k)
			DO UPDATE
				SET v = EXCLUDED.v;
			*/
			_ = db.KvstoreInsertNextPage(ctx, fmt.Sprintf("%d", resp.NextPage))
		} else {
			/* autoquery name: KvstoreInsertEntityTs :exec

			INSERT INTO kvstore (k, v)
			VALUES ('entity-ts', ?)
			ON CONFLICT (k)
			DO UPDATE
				SET v = EXCLUDED.v;
			*/
			_ = db.KvstoreInsertEntityTs(ctx, fmt.Sprintf("%d", time.Now().Unix()))
		}

		return resp.NextPage != 0, nil
	}
}
