package migration

import (
	"database/sql"
	"fmt"
	"github.com/pressly/goose"
	uuid "github.com/satori/go.uuid"
	"time"
)

func init() {
	goose.AddMigration(Up20200616113658, Down20200616113658)
}

func Up20200616113658(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS float_manager_params (
		id varchar(36) NOT NULL, 
		created_at timestamp NULL, 
		updated_at timestamp NULL,   
		min_percent_max_user_balance decimal(64,2) NOT NULL,
		max_percent_max_user_balance decimal(64,2) NOT NULL,
		min_percent_total_user_balance decimal(64,2) NOT NULL,
		average_percent_total_user_balance decimal(64,2) NOT NULL,
		max_percent_total_user_balance decimal(64,2) NOT NULL,
		percent_minimum_trigger_level decimal(64,2) NOT NULL,
		percent_maximum_trigger_level decimal(64,2) NOT NULL,
	
		PRIMARY KEY (id)
	);`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(fmt.Sprintf("INSERT into float_manager_params (id, created_at, updated_at, min_percent_max_user_balance, max_percent_max_user_balance, min_percent_total_user_balance, average_percent_total_user_balance, max_percent_total_user_balance, percent_minimum_trigger_level, percent_maximum_trigger_level) VALUES ('%s', '%s', '%s', %f, %f, %f, %f, %f, %f, %f)", uuid.NewV4().String(), time.Now().Format("2006-01-02T15:04"), time.Now().Format("2006-01-02T15:04:05"), 0.6, 0.8, 0.3, 0.4, 0.6, 0.1, 0.3))
	if err != nil {
		return err
	}

	return nil
}

func Down20200616113658(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("DROP TABLE IF EXISTS float_manager_params;")
	if err != nil {
		return err
	}
	return nil
}
