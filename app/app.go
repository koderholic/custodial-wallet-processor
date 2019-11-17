package app

import (
	"walletAdapter/config"
	"walletAdapter/utility"
	"database/sql"

	"github.com/gorilla/mux"
)

//App : app struct
type App struct {
	Router *mux.Router
	Logger *utility.Logger
	Config config.Data
	DB     *sql.DB
}
