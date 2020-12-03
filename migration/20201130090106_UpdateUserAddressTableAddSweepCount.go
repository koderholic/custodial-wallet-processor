package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20201130090106, Down20201130090106)
}

func Up20201130090106(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE user_addresses ADD column sweep_count int AFTER `address_provider`")
	if err != nil {
		return err
	}
	_, err2 := tx.Exec("ALTER TABLE user_addresses ADD column next_sweep_time timestamp NULL AFTER `sweep_count`")
	if err2 != nil {
		return err2
	}
	return nil
}

func Down20201130090106(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE user_addresses DROP column sweep_count")
	if err != nil {
		return err
	}
	_, err2 := tx.Exec("ALTER TABLE user_addresses DROP column next_sweep_time")
	if err2 != nil {
		return err2
	}
	return nil
}
