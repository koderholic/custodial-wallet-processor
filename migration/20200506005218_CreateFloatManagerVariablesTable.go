package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200506005218, Down20200506005218)
}

func Up20200506005218(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS float_manager_variables (
		id varchar(36) NOT NULL, 
		created_at timestamp NULL, 
		updated_at timestamp NULL,  
		residual_amount decimal(64,18) NOT NULL, 
		asset_symbol varchar(255), 
		total_user_balance decimal(64,18) NOT NULL, 
		percentage_user_balance decimal(64,18) NOT NULL, 
		deposit_sum decimal(64,18) NOT NULL, 
		withdrawal_sum decimal(64,18) NOT NULL, 
		float_on_chain_balance decimal(64,18) NOT NULL, 
		minimum_float_range decimal(64,18) NOT NULL, 
		maximum_float_range decimal(64,18) NOT NULL, 
		deficit decimal(64,18) NOT NULL, 
		last_run_time timestamp NULL, 
		action varchar(255), 
	
		PRIMARY KEY (id)
	);`)
	if err != nil {
		return err
	}
	return nil
}

func Down20200506005218(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("DROP TABLE IF EXISTS float_manager;")
	if err != nil {
		return err
	}
	return nil
}
