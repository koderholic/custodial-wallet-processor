package services

import (
	uuid "github.com/satori/go.uuid"
	"golang.org/x/tools/go/ssa/interp/testdata/src/fmt"
	"wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/model"
	"wallet-adapter/utility"
)

func initHotWallet(repository database.IRepository, logger *utility.Logger, configuration config.Data, userID uuid.UUID, symbol string, serviceErr interface{}) {

	//APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))

	var externalServiceErr model.ServicesRequestErr

	// 1. First check if hot wallet addresses haven't been generated yet.
	// 2. If not yet, generate a new address per each asset supported.

	for _, asset := range config.SupportedAssets {

		address, err := GenerateAddress(logger, configuration, uuid.NewV4(), asset, externalServiceErr)

		if err != nil {

		}

		fmt.Printf("Address generated %s", address)

		//repository.Create()

	}

}

// Get the Bundle hot wallet address corresponding to a certain asset
func getHotWalletAddressFor(asseSymbol string) string {

	// 1. Fetch from hot wallet asset table address that belongs to this assetSymbol and still active

	return ""
}
