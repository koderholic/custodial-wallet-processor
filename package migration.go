package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200327170303, Down20200327170303)
}

func Up20200327170303(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE user_assets ADD coin_type varchar(255);")
	if err != nil {
		return err
	}
	return nil
}

func Down20200327170303(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE user_assets DROP COLUMN coin_type")
	if err != nil {
		return err
	}
	return nil
}
