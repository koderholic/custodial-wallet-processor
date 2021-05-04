package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210415154015, Down20210415154015)
}

func Up20210415154015(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE transactions ADD network varchar(150);")
	if err != nil {
		return err
	}
	_, err1 := tx.Exec("UPDATE transactions set network='ERC20' where asset_symbol IN ('ETH','USDT','LINK');")
	if err1 != nil {
		return err1
	}
	_, err2 := tx.Exec("UPDATE transactions set network='BEP2' where asset_symbol IN ('BNB','BUSD','WRX');")
	if err2 != nil {
		return err2
	}
	_, err3 := tx.Exec("UPDATE transactions set network='TRC20' where asset_symbol IN ('TRX');")
	if err3 != nil {
		return err3
	}
	_, err4 := tx.Exec("UPDATE transactions set network='BTC' where asset_symbol IN ('BTC');")
	if err4 != nil {
		return err4
	}
	_, err5 := tx.Exec("UPDATE transactions set network='BCH' where asset_symbol IN ('BCH');")
	if err5 != nil {
		return err5
	}
	return nil
}

func Down20210415154015(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE transactions DROP COLUMN network;")
	if err != nil {
		return err
	}
	return nil
}
