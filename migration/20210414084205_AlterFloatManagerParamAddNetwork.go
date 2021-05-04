package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210414084205, Down20210414084205)
}

func Up20210414084205(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE float_manager_params ADD network VARCHAR(150) after `asset_symbol`;")
	if err != nil {
		return err
	}
	_, err1 := tx.Exec("UPDATE float_manager_params set network='ERC20' where asset_symbol IN ('ETH','USDT','LINK');")
	if err1 != nil {
		return err1
	}
	_, err2 := tx.Exec("UPDATE float_manager_params set network='BEP2' where asset_symbol IN ('BNB','BUSD','WRX');")
	if err2 != nil {
		return err2
	}
	_, err3 := tx.Exec("UPDATE float_manager_params set network='TRC20' where asset_symbol IN ('TRX');")
	if err3 != nil {
		return err3
	}
	_, err4 := tx.Exec("UPDATE float_manager_params set network='BTC' where asset_symbol IN ('BTC');")
	if err4 != nil {
		return err4
	}
	_, err5 := tx.Exec("UPDATE float_manager_params set network='BCH' where asset_symbol IN ('BCH');")
	if err5 != nil {
		return err5
	}
	_, err6 := tx.Exec("ALTER table float_manager_params ADD INDEX idx_asset_symbol (asset_symbol);")
	if err6 != nil {
		return err6
	}
	_, err7 := tx.Exec("ALTER table float_manager_params ADD INDEX idx_network (network);")
	if err7 != nil {
		return err7
	}
	return nil
}

func Down20210414084205(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE float_manager_params DROP COLUMN network;")
	if err != nil {
		return err
	}
	return nil
}
