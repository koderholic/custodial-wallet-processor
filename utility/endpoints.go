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
	case "createAddress":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.KeyManagementService,
			Action:   "/address/create",
		}
	case "signTransaction":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.KeyManagementService,
			Action:   "/sign-transaction",
		}
	case "broadcastTransaction":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.CryptoAdapterService,
			Action:   "/broadcast-transaction",
		}
	default:
		return MetaData{}
	}
}
