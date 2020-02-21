package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200220214254, Down20200220214254)
}

func Up20200220214254(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("CREATE TABLE IF NOT EXISTS `denominations` (`id` varchar(36) NOT NULL, `created_at` timestamp NULL, `updated_at` timestamp NULL, `name` varchar(255), `symbol` varchar(255) NOT NULL, `token_type` varchar(255) NOT NULL, `decimal` int, `is_enabled` tinyint(1) DEFAULT 1, PRIMARY KEY (id), CONSTRAINT uix_denominations_symbol UNIQUE (symbol), INDEX isEnabled (is_enabled),INDEX symbol (symbol));")
    if err != nil {
        return err
    }
	return nil
}

func Down20200220214254(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("DROP TABLE IF EXISTS denominations;")
    if err != nil {
        return err
    }
	return nil
}
