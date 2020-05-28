package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200528150944, Down20200528150944)
}

func Up20200528150944(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE chain_transactions DROP INDEX uix_trasaction_hash_recipient_address;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE chain_transactions ADD CONSTRAINT uix_transaction_hash_recipient_address UNIQUE (transaction_hash);")
	if err != nil {
		return err
	}
	return nil
}

func Down20200528150944(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE chain_transactions ADD CONSTRAINT uix_transaction_hash_recipient_address UNIQUE (transaction_hash, recipient_address);")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE chain_transactions DROP INDEX uix_trasaction_hash_recipient_address;")
	if err != nil {
		return err
	}
	return nil
}
