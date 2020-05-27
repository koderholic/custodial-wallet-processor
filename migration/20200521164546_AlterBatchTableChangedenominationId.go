package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200521164546, Down20200521164546)
}

func Up20200521164546(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE batch_requests Change denomination_id asset_symbol varchar(100);")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE batch_requests DROP index denomination_id;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE batch_requests Change status status VARCHAR(100) NOT NULL DEFAULT 'WAIT_MODE';")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE batch_requests Change records no_of_records int;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE batch_requests ADD INDEX (status,asset_symbol);")
	if err != nil {
		return err
	}
	return nil
}

func Down20200521164546(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE batch_requests Change asset_symbol denomination_id varchar(36);")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE batch_requests Change status status VARCHAR(100)  NOT NULL DEFAULT 'PENDING';")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE batch_requests Change no_of_records records int;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE batch_requests DROP INDEX (status,asset_symbol);")
	if err != nil {
		return err
	}
	return nil
}
