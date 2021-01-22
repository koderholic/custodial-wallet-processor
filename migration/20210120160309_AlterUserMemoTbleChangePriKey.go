package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210120160309, Down20210120160309)
}

func Up20210120160309(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`ALTER TABLE user_memos DROP PRIMARY KEY`)
	if err != nil {
		return err
	}
	_, err2 := tx.Exec(`ALTER TABLE user_memos ADD PRIMARY KEY (memo);`)
	if err2 != nil {
		return err
	}
	return nil
}

func Down20210120160309(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec(`ALTER TABLE user_memos DELETE PRIMARY KEY memo;`)
	if err != nil {
		return err
	}
	_, err2 := tx.Exec(`ALTER TABLE user_memos ADD PRIMARY KEY (user_id);`)
	if err2 != nil {
		return err
	}
	return nil
}