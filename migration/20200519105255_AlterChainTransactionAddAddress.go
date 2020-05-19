package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200519105255, Down20200519105255)
}

func Up20200519105255(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE chain_transactions ADD recipient_address varchar(100);")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE chain_transactions ADD CONSTRAINT uix_transaction_hash_recipient_address UNIQUE (transaction_hash, recipient_address);")
	if err != nil {
		return err
	}
	return nil
}

func Down20200519105255(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE chain_transactions DROP COLUMN recipient_address;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE chain_transactions DROP INDEX uix_trasaction_hash_recipient_address;")
	if err != nil {
		return err
	}
	return nil
}
