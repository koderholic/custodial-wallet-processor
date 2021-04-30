package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210331092754, Down20210331092754)
}

func Up20210331092754(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE user_addresses ADD COLUMN `network` VARCHAR(100);")
	if err != nil {
		return err
	}

	_, err1 := tx.Exec("UPDATE user_addresses set network='ERC20' where asset_id IN (select id from user_assets where denomination_id IN (select id from denominations where coin_type=60));")
	if err1 != nil {
		return err1
	}
	_, err2 := tx.Exec("UPDATE user_addresses set network='BEP2' where asset_id IN (select id from user_assets where denomination_id IN (select id from denominations where coin_type=714));")
	if err2 != nil {
		return err2
	}
	_, err3 := tx.Exec("UPDATE user_addresses set network='TRC20' where asset_id IN (select id from user_assets where denomination_id IN (select id from denominations where coin_type=195));")
	if err3 != nil {
		return err3
	}
	_, err4 := tx.Exec("UPDATE user_addresses set network='BTC' where asset_id IN (select id from user_assets where denomination_id IN (select id from denominations where coin_type=0));")
	if err4 != nil {
		return err4
	}
	_, err5 := tx.Exec("UPDATE user_addresses set network='BCH' where asset_id IN (select id from user_assets where denomination_id IN (select id from denominations where coin_type=145));")
	if err5 != nil {
		return err5
	}
	return nil
}

func Down20210331092754(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE user_addresses DROP COLUMN network;")
	if err != nil {
		return err
	}
	return nil
}
