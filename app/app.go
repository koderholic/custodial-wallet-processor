package app

import (
	"wallet-adapter/config"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
)

//App : app struct
type App struct {
	Router *mux.Router
	Logger *utility.Logger
	Config config.Data
	DB     *gorm.DB
}
