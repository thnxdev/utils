//go:generate autoquery
package wkrrepos

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/alecthomas/errors"
	"github.com/shurcooL/githubv4"
	wkrghsponsor "github.com/thnxdev/utils"
	"github.com/thnxdev/utils/database"
	"github.com/thnxdev/utils/utils/httpgh"
	"github.com/thnxdev/utils/utils/log"
	"github.com/thnxdev/utils/workers"
	"golang.org/x/oauth2"
)

func init() {
	workers.Register(New)
}

func New(
	db *database.DB,
	ghAccesstoken wkrghsponsor.GhAccessToken,
	donorEntities []wkrghsponsor.Entity,
) workers.Worker {
	return func(ctx context.Context) (bool, error) {
		/* autoquery name: GetRepos :one

		SELECT owner_name, repo_name, cursor_manifest, cursor_dep
		FROM repos
		WHERE animate_ts < last_ts
		LIMIT 1;
		*/
		row, err := db.GetRepos(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// If entities is finished and there are no more rows
				// then this worker should finish as well

				/* autoquery name: GetAreReposFinished :one

				WITH d AS (VALUES(1))
				SELECT kv.v IS NOT NULL AS is_finished
				FROM d
				LEFT JOIN kvstore kv ON k = 'entity-ts';
				*/
				isFinished, err := db.GetAreReposFinished(ctx)
				if err != nil {
					return false, nil
				}
				if isFinished {
					return false, workers.ErrDone
				}
				return false, nil
			}
			return false, errors.Wrap(err, "failed to get repos")
		}

		log.FromContext(ctx).Debugf(
			"processing %s/%s",
			row.OwnerName,
			row.RepoName,
		)

		var mc, dc *string
		if row.CursorManifest.Valid {
			mc = &row.CursorManifest.String
		}
		if row.CursorDep.Valid {
			dc = &row.CursorDep.String
		}

		var q struct {
			Repository struct {
				Name                     string
				DependencyGraphManifests struct {
					Nodes []struct {
						Filename    string
						Depenencies struct {
							Nodes []struct {
								Repository struct {
									Owner struct {
										Sponsorable struct {
											HasSponsorsListing bool
										} `graphql:"... on Sponsorable"`
										RepositoryOwner struct {
											Login string
										} `graphql:"... on RepositoryOwner"`
									}
								}
							}
							PageInfo struct {
								EndCursor   string
								HasNextPage bool
							}
						} `graphql:"dependencies(first: 100, after: $depCursor)"`
					}
					PageInfo struct {
						EndCursor   string
						HasNextPage bool
					}
				} `graphql:"dependencyGraphManifests(first: 1, after: $manifestCursor)"`
			} `graphql:"repository(owner: $owner, name: $name)"`
		}

		var vars map[string]any = map[string]any{
			"owner":          githubv4.String(row.OwnerName),
			"name":           githubv4.String(row.RepoName),
			"manifestCursor": (*githubv4.String)(mc),
			"depCursor":      (*githubv4.String)(dc),
		}

		hctx := context.WithValue(
			ctx,
			oauth2.HTTPClient,
			&http.Client{Transport: httpgh.NewTransport(nil)},
		)

		client := githubv4.NewClient(
			oauth2.NewClient(
				hctx,
				oauth2.StaticTokenSource(&oauth2.Token{
					AccessToken: string(ghAccesstoken),
				}),
			),
		)

		err = client.Query(hctx, &q, vars)
		if err != nil {
			return false, errors.Wrap(err, "failed to query repos")
		}

		var manifetsCursor, depCursor *string
		for _, m := range q.Repository.DependencyGraphManifests.Nodes {
			log.FromContext(ctx).Debugf("processing manifest %s(%d)", m.Filename, len(m.Depenencies.Nodes))
			for _, d := range m.Depenencies.Nodes {
				o := d.Repository.Owner
				if o.Sponsorable.HasSponsorsListing {
					/* autoquery name: InsertDonation :exec

					INSERT INTO donations (sponsor_id, recipient_id, last_ts)
					VALUES (?, ?, ?)
					ON CONFLICT (sponsor_id, recipient_id)
					DO NOTHING;
					*/
					_ = db.InsertDonation(ctx, database.InsertDonationParams{
						SponsorID:   row.OwnerName,
						RecipientID: o.RepositoryOwner.Login,
						LastTs:      time.Now().Unix(),
					})
					log.FromContext(ctx).Debugf("fundable %s", o.RepositoryOwner.Login)
				}
			}
			if m.Depenencies.PageInfo.HasNextPage {
				depCursor = &m.Depenencies.PageInfo.EndCursor
			}
		}

		if q.Repository.DependencyGraphManifests.PageInfo.HasNextPage {
			manifetsCursor = &q.Repository.DependencyGraphManifests.PageInfo.EndCursor
		}

		if depCursor != nil {
			dc := sql.NullString{*depCursor, true}

			/* autoquery name: RepoUpdateCursorDep :exec

			UPDATE repos
			SET cursor_dep = ?
			WHERE owner_name = ? AND repo_name = ?;
			*/
			_ = db.RepoUpdateCursorDep(ctx, database.RepoUpdateCursorDepParams{
				OwnerName: row.OwnerName,
				RepoName:  row.RepoName,
				CursorDep: dc,
			})
		} else if manifetsCursor != nil {
			mc := sql.NullString{*manifetsCursor, true}

			/* autoquery name: RepoUpdateCursorManifest :exec

			UPDATE repos
			SET cursor_manifest = ?
			WHERE owner_name = ? AND repo_name = ?;
			*/
			_ = db.RepoUpdateCursorManifest(ctx, database.RepoUpdateCursorManifestParams{
				OwnerName:      row.OwnerName,
				RepoName:       row.RepoName,
				CursorManifest: mc,
			})
		} else {
			/* autoquery name: RepoUpdateAnimateTs :exec

			UPDATE repos
			SET animate_ts = UNIXEPOCH()
			WHERE owner_name = ? AND repo_name = ?;
			*/
			_ = db.RepoUpdateAnimateTs(ctx, database.RepoUpdateAnimateTsParams{
				OwnerName: row.OwnerName,
				RepoName:  row.RepoName,
			})
		}

		return true, nil
	}
}
