//go:generate autoquery
package importcsv

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/thnxdev/utils"
	"github.com/thnxdev/utils/database"
	"github.com/thnxdev/utils/utils/log"
)

type CmdImportCsv struct {
	FilePath string       `help:"The csv file to import from." type:"path" required:""`
	Entity   utils.Entity `help:"The GitHub entity to import into." required:""`
}

func (c *CmdImportCsv) Run(
	ctx context.Context,
	db *database.DB,
) error {
	log.FromContext(ctx).Infof("opening %s for import", c.FilePath)

	file, err := os.Open(c.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open %s", c.FilePath)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	lines, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read %s", c.FilePath)
	}

	if len(lines) == 0 {
		return errors.New("file is empty")
	}

	// first line is header
	for _, line := range lines[1:] {
		ghUsername := line[0]
		log.FromContext(ctx).Infof("adding %s", ghUsername)
		_ = db.InsertDonation(ctx, database.InsertDonationParams{
			SponsorID:   string(c.Entity),
			RecipientID: ghUsername,
			LastTs:      time.Now().Unix(),
		})
	}

	return nil
}
