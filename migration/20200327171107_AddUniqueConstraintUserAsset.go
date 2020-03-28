package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200327171107, Down20200327171107)
}

func Up20200327171107(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE user_assets ADD CONSTRAINT denomination_userId UNIQUE (denomination_id, user_id);")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE user_assets DROP FOREIGN KEY user_assets_denomination_id_denominations_id_foreign;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE user_assets ADD CONSTRAINT user_assets_denomination_id_foreign FOREIGN KEY (denomination_id) REFERENCES denominations (id) ON DELETE RESTRICT ON UPDATE NO ACTION;")
	if err != nil {
		return err
	}

	return nil
}

func Down20200327171107(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE user_assets DROP CONSTRAINT denomination_userId;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE user_assets ADD CONSTRAINT  user_assets_denomination_id_denominations_id_foreign FOREIGN KEY (denomination_id) REFERENCES denominations (id) ON DELETE NO ACTION ON UPDATE NO ACTION;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE user_assets DROP FOREIGN KEY user_assets_denomination_id_foreign;")
	if err != nil {
		return err
	}
	return nil
}
