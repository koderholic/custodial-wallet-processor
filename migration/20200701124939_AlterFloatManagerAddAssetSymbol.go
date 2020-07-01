package migration

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pressly/goose"
	uuid "github.com/satori/go.uuid"
)

func init() {
	goose.AddMigration(Up20200701124939, Down20200701124939)
}

func Up20200701124939(tx *sql.Tx) error {
	// This code is executed when the migration is applied.

	_, err := tx.Exec("ALTER TABLE float_manager_params ADD COLUMN asset_symbol VARCHAR(100);")
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM float_manager_params;")
	if err != nil {
		return err
	}

	supportedAssets := []string{"BTC", "BNB", "BUSD", "ETH"}
	for _, asset := range supportedAssets {
		_, err := tx.Exec(fmt.Sprintf("INSERT into float_manager_params (id, created_at, updated_at, min_percent_max_user_balance, max_percent_max_user_balance, min_percent_total_user_balance, average_percent_total_user_balance, max_percent_total_user_balance, percent_minimum_trigger_level, percent_maximum_trigger_level, asset_symbol) VALUES ('%s', '%s', '%s', %f, %f, %f, %f, %f, %f, %f, '%s')", uuid.NewV4().String(), time.Now().Format("2006-01-02T15:04"), time.Now().Format("2006-01-02T15:04:05"), 0.6, 0.8, 0.3, 0.4, 0.6, 0.8, 0.3, asset))
		if err != nil {
			return err
		}
	}

	return nil
}

func Down20200701124939(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE float_manager_params DROP COLUMN asset_symbol;")
	if err != nil {
		return err
	}
	return nil
}
