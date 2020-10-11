package test

import (
	"strconv"
	"wallet-adapter/dto"
	"wallet-adapter/services"
	"wallet-adapter/utility"
	"wallet-adapter/utility/appError"

	"github.com/stretchr/testify/assert"
)

func (s *Suite) Test_GetTransactionStatus_returns_right_status_for_tx() {
	userAsset := testUserAssets1[3]
	broadcastedTX := []dto.AllAddressResponse{
		{
			Type: "Legacy",
			Data: "32DeyC6iPWVwDDngAAGJoYRqWJK6aPPEfE",
		},
		{
			Type: "Segwit",
			Data: "bc1qug3tpy7ppj6um44sauq8vr6e55ygsynlwm02ve",
		},
	}
	

	TransactionService := services.NewTransactionService(authCache, s.Config, &testUserAssetRepository)
	assetAddresses, err := TransactionService.GetTransactionStatus(true, broadcastedTX, transaction, transactionQueue)

	assert.Equal(s.T(), nil, err, "Expected GetAddresses to not return error")
	assert.Equal(s.T(), 2, len(assetAddresses), "GetAddresses did not generate the complete addresses for asset")
	if len(assetAddresses) > 0 {
		assert.Equal(s.T(), "bc1qug3tpy7ppj6um44sauq8vr6e55ygsynlwm02ve", assetAddresses[0].Address, "GetAddresses did not sort by default address")
	}
}