package database

import (
	"walletAdapter/config"
	"walletAdapter/utility"
	"database/sql"
)

//Database : database struct
type Database struct {
	Logger *utility.Logger
	Config config.Data
	DB     *sql.DB
}
