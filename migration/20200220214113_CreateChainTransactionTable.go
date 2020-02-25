package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200220214113, Down20200220214113)
}

func Up20200220214113(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS chain_transactions (
		id varchar(36) NOT NULL, 
		created_at timestamp NULL, 
		updated_at timestamp NULL, 
		status tinyint(1) DEFAULT 0 NOT NULL, 
		batch_id varchar(36), 
		transaction_hash varchar(255) NOT NULL, 
		block_height bigint, 
		transaction_fee varchar(255),
	
		PRIMARY KEY (id), 
		INDEX idx_chain_transactions_status (status), 
		INDEX batch_id (batch_id)
	);`)
    if err != nil {
        return err
    }
	return nil
}

func Down20200220214113(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("DROP TABLE IF EXISTS chain_transactions;")
    if err != nil {
        return err
    }
	return nil
}
