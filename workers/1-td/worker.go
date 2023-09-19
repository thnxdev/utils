//go:generate autoquery
package wkrtd

//
// This worker continuously checks api.thanks.dev/v1/deps to get the user's latest
// list of GH dependencies.
//

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	wkrghsponsor "github.com/thnxdev/wkr-gh-sponsor"
	"github.com/thnxdev/wkr-gh-sponsor/database"
	"github.com/thnxdev/wkr-gh-sponsor/utils/log"
	"github.com/thnxdev/wkr-gh-sponsor/workers"
)

func New(
	db *database.DB,
	tdApiUrl wkrghsponsor.TdApiUrl,
	tdApiKey wkrghsponsor.TdApiKey,
	donorEntities []wkrghsponsor.Entity,
) workers.Worker {
	return func(ctx context.Context) error {
		logger := log.FromContext(ctx)

		req, err := http.NewRequest("GET", string(tdApiUrl), nil)
		if err != nil {
			logger.WithError(err).Error("request failed")
			return err
		}
		req.Header.Set("api-key", string(tdApiKey))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			logger.WithError(err).Error("request failed")
			return err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.Errorf("td api call failed with status code %d", resp.StatusCode)
			return errors.New("td api call failed")
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.WithError(err).Error("failed to read repsonse")
			return err
		}

		type dependency struct {
			Login    string `json:"login"`
			Entities []int  `json:"entities"`
		}

		var data struct {
			Entities     []string     `json:"entities"`
			Dependencies []dependency `json:"dependencies"`
		}

		err = json.Unmarshal(body, &data)
		if err != nil {
			logger.WithError(err).Error("failed to parse payload")
			return err
		}

		entityIndex := map[int]string{}
		for i, ent := range data.Entities {
			entityIndex[i] = ent
		}

		donorEntityIndex := map[string]int{}
		for i, ent := range donorEntities {
			donorEntityIndex[string(ent)] = i
		}

		for _, dep := range data.Dependencies {
			for _, ent := range dep.Entities {
				entName, ok := entityIndex[ent]
				if !ok {
					continue
				}

				donorEntIdx, ok := donorEntityIndex[entName]
				if !ok {
					donorEntIdx = 0
				}

				donorName := donorEntities[donorEntIdx]

				/* autoquery name: InsertDonation :exec

				INSERT INTO donations (sponsor_id, recipient_id, last_ts)
				VALUES (?, ?, UNIXEPOCH())
				ON CONFLICT (sponsor_id, recipient_id)
				DO UPDATE
					SET last_ts = EXCLUDED.last_ts;
				*/
				err := db.InsertDonation(ctx, database.InsertDonationParams{
					SponsorID:   string(donorName),
					RecipientID: dep.Login,
				})
				if err != nil {
					logger.WithError(err).Error("failed to insert donation")
					return err
				}
			}
		}
		return nil
	}
}
