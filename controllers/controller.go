package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"wallet-adapter/config"
	"wallet-adapter/utility"
)

//Controller : Controller struct
type Controller struct {
	Logger *utility.Logger
	Config config.Data
	DB     *sql.DB
}

//Ping : Ping function
func (c *Controller) Ping(w http.ResponseWriter, r *http.Request) {

	apiResponse := utility.NewResponse()

	c.Logger.Info("Ping request successful! Server is up and listening")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse.PlainSuccess(utility.SUCCESS, "Ping request successful! Server is up and listening"))
}
