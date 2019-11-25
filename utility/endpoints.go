package utility

import (
	"wallet-adapter/config"
)

type MetaData struct {
	Type, Endpoint, Action string
}

func GetRequestMetaData(request string, Config config.Data) MetaData {
	switch request {
	default:
		return MetaData{}
	}
}
