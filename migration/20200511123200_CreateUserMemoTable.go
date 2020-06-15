package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200511123200, Down20200511123200)
}

func Up20200511123200(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS user_memos (
			user_id varchar(36) NOT NULL,
			memo varchar(15) NOT NULL,
		
			PRIMARY KEY (user_id), 
			CONSTRAINT uix_user_memos_memo UNIQUE (memo)
		);`)
	if err != nil {
		return err
	}
	return nil
}

func Down20200511123200(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("DROP TABLE IF EXISTS user_memos;")
	if err != nil {
		return err
	}
	return nil
}
