package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210331092510, Down20210331092510)
}

func Up20210331092510(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE hot_wallet_assets ADD COLUMN `network` VARCHAR(100) AFTER `reserved_balance`;")
	if err != nil {
		return err
	}
	_, err1 := tx.Exec("UPDATE hot_wallet_assets set network='ERC20' where asset_symbol IN ('ETH','USDT','LINK');")
	if err1 != nil {
		return err1
	}
	_, err2 := tx.Exec("UPDATE hot_wallet_assets set network='BEP2' where asset_symbol IN ('BNB','BUSD','WRX');")
	if err2 != nil {
		return err2
	}
	_, err3 := tx.Exec("UPDATE hot_wallet_assets set network='TRC20' where asset_symbol IN ('TRX');")
	if err3 != nil {
		return err3
	}
	_, err4 := tx.Exec("UPDATE hot_wallet_assets set network='BTC' where asset_symbol IN ('BTC');")
	if err4 != nil {
		return err4
	}
	_, err5 := tx.Exec("UPDATE hot_wallet_assets set network='BCH' where asset_symbol IN ('BCH');")
	if err5 != nil {
		return err5
	}
	_, err6 := tx.Exec("ALTER table hot_wallet_assets ADD INDEX idx_asset_symbol (asset_symbol);")
	if err6 != nil {
		return err6
	}
	_, err7 := tx.Exec("ALTER table hot_wallet_assets ADD INDEX idx_network (network);")
	if err7 != nil {
		return err7
	}
	_, err8 := tx.Exec("ALTER table hot_wallet_assets DROP INDEX uix_hot_wallet_assets_asset_symbol;")
	if err8 != nil {
		return err8
	}
	_, err9 := tx.Exec("ALTER table hot_wallet_assets ADD CONSTRAINT uix_hot_wallet_assets_asset_symbol_network UNIQUE (asset_symbol,network);")
	if err9 != nil {
		return err9
	}
	return nil
}

func Down20210331092510(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE hot_wallet_assets DROP COLUMN network;")
	if err != nil {
		return err
	}
	return nil
}
