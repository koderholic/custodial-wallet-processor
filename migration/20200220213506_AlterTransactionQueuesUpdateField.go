package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200220213506, Down20200220213506)
}

func Up20200220213506(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE transaction_queues MODIFY value BIGINT(20);")
    if err != nil {
        return err
    }
	return nil
}

func Down20200220213506(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE transaction_queues MODIFY value DECIMAL(64,18);")
    if err != nil {
        return err
    }
	return nil
}
