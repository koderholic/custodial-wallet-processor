package database

import (
	"wallet-adapter/config"
	"wallet-adapter/utility"
	"database/sql"
)

//Database : database struct
type Database struct {
	Logger *utility.Logger
	Config config.Data
	DB     *sql.DB
}
