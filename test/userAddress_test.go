package test

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"wallet-adapter/dto"
	"wallet-adapter/services"
	"wallet-adapter/utility"
)

func (s *Suite) Test_GetAddresses_returns_all_addresses_for_asset_and_symbol() {
	userAsset := testUserAssets1[3]

	UserAddressService := services.NewUserAddressService(authCache, s.Config, &testUserAssetRepository)
	assetAddresses, err := UserAddressService.GetAddresses(userAsset)

	assert.Equal(s.T(), nil, err, "Expected GetAddresses to not return error")
	assert.Equal(s.T(), 2, len(assetAddresses), "GetAddresses did not generate the complete addresses for asset")
	//if len(assetAddresses) > 0 {
	//	assert.Equal(s.T(), "bc1qug3tpy7ppj6um44sauq8vr6e55ygsynlwm02ve", assetAddresses[0].Address, "GetAddresses did not sort by default address")
	//}
}

func (s *Suite) Test_GetV1Address_returns_empty_for_non_existing_address() {
	userAsset := testUserAssets2[2]

	UserAddressService := services.NewUserAddressService(authCache, s.Config, &testUserAssetRepository)
	assetAddress, err := UserAddressService.GetV1Address(userAsset)

	assert.Equal(s.T(), nil, err, "Expected GetV1Address to not return error")
	assert.Equal(s.T(), "", assetAddress, "GetV1Address should return empty")
}

func (s *Suite) Test_GetV1Address_returns_address_for_existing_address() {
	userAsset := testUserAssets1[2]

	UserAddressService := services.NewUserAddressService(authCache, s.Config, &testUserAssetRepository)
	assetAddress, err := UserAddressService.GetV1Address(userAsset)

	assert.Equal(s.T(), nil, err, "Expected GetV1Address to not return error")
	assert.Equal(s.T(), "0xce4B800c0aB49Dda535BCe18F87f81D13f142A3C", assetAddress, "GetV1Address should return empty")
}

func (s *Suite) Test_CreateV1Address_returns_newly_created_address() {
	userAsset := testUserAssets2[2]
	keyManagementAddressResponseMock := []dto.AllAddressResponse{
		{
			Type: "",
			Data: "0x4F499d193346E9cb602dA5B2A8ffd45f37AFD842",
		},
	}

	UserAddressService := services.NewUserAddressService(authCache, s.Config, &testUserAssetRepository)
	assetAddress, err := UserAddressService.CreateV1Address(userAsset, keyManagementAddressResponseMock)

	assert.Equal(s.T(), nil, err, "Expected GetV1Address to not return error")
	assert.Equal(s.T(), "0x4F499d193346E9cb602dA5B2A8ffd45f37AFD842", assetAddress, "assetAddress should return empty")
}

func (s *Suite) Test_AssetAddresses_returns_error_when_deposit_disabled() {
	UserAddressService := services.NewUserAddressService(authCache, s.Config, &testUserAssetRepository)
	assetAddresses, err := UserAddressService.GetAddressesFor(testUserAssets1[4].ID)

	assert.NotEqual(s.T(), nil, err, "Expected GetAddressesFor to return error if deposit is not supported")
	assert.Equal(s.T(), 0, len(assetAddresses.Addresses), "assetAddress should return empty")
}

func (s *Suite) Test_GetV2Address_returns_v1address_for_existing_address() {
	userAssetBNB := testUserAssets1[0]
	userAssetBUSD := testUserAssets1[1]

	UserAddressService := services.NewUserAddressService(authCache, s.Config, &testUserAssetRepository)
	assetAddressBNB, err := UserAddressService.GetV2Address(userAssetBNB)
	assetAddressBUSD, err := UserAddressService.GetV2Address(userAssetBUSD)

	assert.Equal(s.T(), nil, err, "Expected GetV2Address to not nerror")
	assert.Equal(s.T(), "bnb10f7jqrvg3d978cgtsqydtlk20y992yeapjzd3a", assetAddressBNB.Address, "assetAddress should return asset v2 address")
	assert.Equal(s.T(), "639469678", assetAddressBNB.Memo, "assetAddress should return asset memo")
	assert.Equal(s.T(), assetAddressBUSD.Memo, assetAddressBNB.Memo, "assetAddress should return same memo for same coinType")
	assert.Equal(s.T(), assetAddressBUSD.Address, assetAddressBNB.Address, "assetAddress should return same address for same coinType")
}

func (s *Suite) Test_GetV2Address_returns_v1address_for_nonexisting_address() {
	userAssetBNB := testUserAssets2[0]

	UserAddressService := services.NewUserAddressService(authCache, s.Config, &testUserAssetRepository)
	assetAddressBNB, err := UserAddressService.GetV2Address(userAssetBNB)

	assert.Equal(s.T(), nil, err, "Expected GetV2Address to not nerror")
	assert.Equal(s.T(), "bnb10f7jqrvg3d978cgtsqydtlk20y992yeapjzd3a", assetAddressBNB.Address, "assetAddress should return the shared address")
	assert.NotEqual(s.T(), "", assetAddressBNB.Memo, "assetAddress should not return empty asset memo")
}

func (s *Suite) Test_CheckV2Address_returns_true_for_v2addresses() {
	v2Address := "bnb10f7jqrvg3d978cgtsqydtlk20y992yeapjzd3a"

	UserAddressService := services.NewUserAddressService(authCache, s.Config, &testUserAssetRepository)
	isV2Address, err := UserAddressService.CheckV2Address(v2Address)

	assert.Equal(s.T(), nil, err, "Expected CheckV2Address to not error")
	assert.Equal(s.T(), true, isV2Address, "Expected CheckV2Address to return true")
}

func (s *Suite) Test_CheckV2Address_returns_false_for_non_v2addresses() {
	v2Address := "0x4F499d193346E9cb602dA5B2A8ffd45f37AFD842"

	UserAddressService := services.NewUserAddressService(authCache, s.Config, &testUserAssetRepository)
	isV2Address, err := UserAddressService.CheckV2Address(v2Address)

	assert.Equal(s.T(), nil, err, "Expected CheckV2Address to not error")
	assert.Equal(s.T(), false, isV2Address, "Expected CheckV2Address to return true")
}

func (s *Suite) Test_TransformAddressesResponse_returns_correct_format() {
	inputAddressResponse := []dto.AllAddressResponse{
		{
			Data: "0x4F499d193346E9cb602dA5B2A8ffd45f37AFD842",
		},
	}
	expectedAddressResponse := []dto.AssetAddress{
		{
			Address: "0x4F499d193346E9cb602dA5B2A8ffd45f37AFD842",
		},
	}

	UserAddressService := services.NewUserAddressService(authCache, s.Config, &testUserAssetRepository)
	transformAddressesResponse := UserAddressService.TransformAddressesResponse(inputAddressResponse)
	assert.Equal(s.T(), expectedAddressResponse, transformAddressesResponse, "assetAddress should return empty")
}

func (s *Suite) Test_FixedMemoGenerationLength() {
	for i := 0; i <= 100; i++ {
		memo := strconv.Itoa(utility.RandNo(100000000, 999999999))
		assert.Equal(s.T(), len(memo), 9)
	}
}

func (s *Suite) Test_RandomMemoGeneration() {
	memos := map[string]bool{}
	for i := 0; i <= 10000; i++ {
		memo := strconv.Itoa(utility.RandNo(100000000, 999999999))
		assert.Equal(s.T(), memos[memo], false)
		memos[memo] = true
	}
}
