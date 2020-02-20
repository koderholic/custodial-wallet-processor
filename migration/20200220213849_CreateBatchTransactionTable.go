package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200220213849, Down20200220213849)
}

func Up20200220213849(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS batch_requests (
		id VARCHAR(36),
		denomination_id VARCHAR(36)  NOT NULL,
		status VARCHAR(100)  NOT NULL DEFAULT 'PENDING',
		created_at DATETIME NULL,
		updated_at DATETIME NULL,
		date_completed DATETIME NULL,
		date_of_processing DATETIME NULL,
		records int,

        PRIMARY KEY (id),
        INDEX status (status), 
        INDEX denomination_id (denomination_id)
	);`)
    if err != nil {
        return err
    }
	return nil
}

func Down20200220213849(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("DROP TABLE IF EXISTS batch_requests;")
    if err != nil {
        return err
    }
	return nil
}
