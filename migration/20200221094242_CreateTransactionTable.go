package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200221094242, Down20200221094242)
}

func Up20200221094242(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS transactions (
		id varchar(36) NOT NULL, 
		created_at timestamp NULL, 
		updated_at timestamp NULL, 
		initiator_id varchar(36) NOT NULL, 
		transaction_reference varchar(255) NOT NULL, 
		transaction_type varchar(255) DEFAULT 'OFFCHAIN' NOT NULL, 
		transaction_status varchar(255) DEFAULT 'PENDING' NOT NULL, 
		transaction_tag varchar(255) DEFAULT 'SELL' NOT NULL, 
		processing_type varchar(255) DEFAULT 'SINGLE' NOT NULL, 
		batch_id varchar(36), 
		asset_symbol varchar(255),
		transaction_start_date timestamp NULL, 
		transaction_end_date timestamp NULL, 
		recipient_id varchar(36), 
		payment_reference varchar(255) NOT NULL, 
		memo varchar(255) NOT NULL, 
		value decimal(64,18) NOT NULL, 
		previous_balance decimal(64,18) NOT NULL, 
		available_balance decimal(64,18) NOT NULL, 
		on_chain_tx_id varchar(36), 
		debit_reference varchar(255), 
		swept_status tinyint(1) DEFAULT 0 NOT NULL, 
	
		PRIMARY KEY (id), 
		CONSTRAINT uix_transactions_transaction_reference UNIQUE (transaction_reference), 
		CONSTRAINT uix_transactions_payment_reference UNIQUE (payment_reference), 
		CONSTRAINT transactions_recipient_id_recipient_id_foreign FOREIGN KEY (recipient_id) REFERENCES user_assets (id) ON DELETE NO ACTION ON UPDATE NO ACTION, 
		INDEX initiator_id (initiator_id), INDEX transaction_status (transaction_status)
	);`)
	if err != nil {
		return err
	}
	return nil
}

func Down20200221094242(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("DROP TABLE IF EXISTS transactions;")
	if err != nil {
		return err
	}
	return nil
}
