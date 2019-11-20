package controllers

import (
	"wallet-adapter/database"
)

//Controller : Controller struct
type UserAssetController struct {
	Controller
	repository database.IUserAssetRepository
}
