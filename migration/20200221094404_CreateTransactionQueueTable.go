package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200221094404, Down20200221094404)
}

func Up20200221094404(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS transaction_queues (
		id varchar(36) NOT NULL, 
		created_at timestamp NULL, 
		updated_at timestamp NULL, 
		sender varchar(255), 
		recipient varchar(255) NOT NULL, 
		value BIGINT(20) NOT NULL, 
		asset_symbol varchar(255) NOT NULL, 
		debit_reference varchar(150) NOT NULL, 
		transaction_id varchar(36) NOT NULL, 
		transaction_status varchar(255) DEFAULT 'PENDING' NOT NULL, 
		memo varchar(300), PRIMARY KEY (id), 
			
		CONSTRAINT uix_transaction_queues_debit_reference UNIQUE (debit_reference),  
		CONSTRAINT transaction_queues_transaction_id_transaction_id_foreign FOREIGN KEY (transaction_id) REFERENCES transactions (id) ON DELETE CASCADE ON UPDATE CASCADE, 
		CONSTRAINT transaction_queues_debit_reference_debit_reference_foreign FOREIGN KEY (debit_reference) REFERENCES transactions (transaction_reference) ON DELETE NO ACTION ON UPDATE NO ACTION, 
		INDEX transaction_status (transaction_status), 
		INDEX transaction_id (transaction_id)
	);`)
	if err != nil {
		return err
	}
	return nil
}

func Down20200221094404(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("DROP TABLE IF EXISTS transactions;")
	if err != nil {
		return err
	}
	return nil
}
