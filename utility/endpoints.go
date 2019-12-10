package utility

import (
	"net/http"
	Config "wallet-adapter/config"
)

type MetaData struct {
	Type, Endpoint, Action string
}

// GetRequestMetaData ...
func GetRequestMetaData(request string, config Config.Data) MetaData {
	switch request {
	case "generateToken":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.AuthenticationService,
			Action:   "/services/token",
		}
	case "generateAddress":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.KeyManagementService,
			Action:   "/address/create",
		}
	default:
		return MetaData{}
	}
}
