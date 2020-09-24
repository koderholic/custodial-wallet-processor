package test

import (
	"wallet-adapter/dto"
	"wallet-adapter/services"
	"wallet-adapter/utility/appError"

	uuid "github.com/satori/go.uuid"

	"github.com/stretchr/testify/assert"
)

func (s *Suite) Test_CreateAsset_pass_ForSupportedAssets() {

	denominations := []string{"LINK", "ETH", "BNB"}
	userId, _ := uuid.FromString("a10fce7b-7844-43af-9ed1-e130723a1ea3")

	UserAssetService := services.NewUserAssetService(authCache, s.Config, &testUserAssetRepository, &dto.ExternalServicesRequestErr{})
	createdAsset, err := UserAssetService.CreateAsset(denominations, userId)
	assert.Equal(s.T(), nil, err, "Expected CreateAsset to not return error")
	assert.Equal(s.T(), 3, len(createdAsset), "Assets not completely created")
}

func (s *Suite) Test_CreateAsset_failCompletely_ForNonSupportedAssets() {
	denominations := []string{"LINK", "ETH", "THG"}
	userId, _ := uuid.FromString("a10fce7b-7844-43af-9ed1-e130723a1ea3")

	UserAssetService := services.NewUserAssetService(authCache, s.Config, &testUserAssetRepository, &dto.ExternalServicesRequestErr{})
	createdAsset, err := UserAssetService.CreateAsset(denominations, userId)
	assert.NotEqual(s.T(), nil, err, "Expected CreateAsset to return error")
	assert.Equal(s.T(), 400, err.(appError.Err).ErrCode, "Expected CreateAsset to return error")
	assert.Equal(s.T(), "ASSET_NOT_SUPPORTED", err.(appError.Err).ErrType, "Expected CreateAsset to return ASSET_NOT_SUPPORTED")
	assert.Equal(s.T(), 0, len(createdAsset), "Assets not completely created")
}

func (s *Suite) Test_CreateAsset_returnsCorrectFields() {
	denominations := []string{"LINK"}
	userId, _ := uuid.FromString("a10fce7b-7844-43af-9ed1-e130723a1ea3")

	expected := dto.Asset{
		UserID:           userId,
		AssetSymbol:      "LINK",
		AvailableBalance: "0",
		Decimal:          18,
	}
	UserAssetService := services.NewUserAssetService(authCache, s.Config, &testUserAssetRepository, &dto.ExternalServicesRequestErr{})
	createdAsset, err := UserAssetService.CreateAsset(denominations, userId)

	assert.Equal(s.T(), nil, err, "Expected CreateAsset to return error")
	assert.Equal(s.T(), 1, len(createdAsset), "Assets not successfully created")
	assert.NotEqual(s.T(), "", createdAsset[0].ID, "Assets not successfully created")
	assert.Equal(s.T(), expected.AssetSymbol, createdAsset[0].AssetSymbol, "Assets not successfully created")
	assert.Equal(s.T(), expected.AvailableBalance, createdAsset[0].AvailableBalance, "Assets not successfully created")
	assert.Equal(s.T(), expected.Decimal, createdAsset[0].Decimal, "Assets not successfully created")
	assert.Equal(s.T(), expected.UserID, createdAsset[0].UserID, "Assets not successfully created")
}

func (s *Suite) Test_FetchAssets_pass_ForExistingUserId() {
	UserAssetService := services.NewUserAssetService(authCache, s.Config, &testUserAssetRepository, &dto.ExternalServicesRequestErr{})
	userAssets, err := UserAssetService.FetchAssets(testUserId1)

	assert.Equal(s.T(), nil, err, "Expected CreateAsset to return error")
	assert.Equal(s.T(), 5, len(userAssets), "Assets not successfully crereturnedated")
	assert.Equal(s.T(), testDenominations[0].AssetSymbol, userAssets[0].AssetSymbol, "Assets not successfully created")
	assert.Equal(s.T(), "0", userAssets[0].AvailableBalance, "Assets not successfully created")
	assert.Equal(s.T(), testDenominations[0].Decimal, userAssets[0].Decimal, "Assets not successfully created")
	assert.Equal(s.T(), testUserId1, userAssets[0].UserID, "Assets not successfully created")
}

func (s *Suite) Test_FetchAssets_Fails_ForNonExistingUserId() {
	nonExistingUserId, _ := uuid.FromString("a10fce7b-7844-43af-9ed1-e130723a0003")
	UserAssetService := services.NewUserAssetService(authCache, s.Config, &testUserAssetRepository, &dto.ExternalServicesRequestErr{})
	userAssets, err := UserAssetService.FetchAssets(nonExistingUserId)

	assert.NotEqual(s.T(), nil, err, "Expected FetchAssets to return error")
	assert.Equal(s.T(), 400, err.(appError.Err).ErrCode, "Expected FetchAssets to return error")
	assert.Equal(s.T(), "RECORD_NOT_FOUND", err.(appError.Err).ErrType, "Expected FetchAssets to return RECORD_NOT_FOUND")
	assert.Equal(s.T(), 0, len(userAssets), "Assets should not exist")
}

func (s *Suite) Test_GetAssetById_pass_ForExistingAssetId() {

	denominations := []string{"BNB"}
	userId, _ := uuid.FromString("a10fce7b-7844-43af-9ed1-e130723a1ea3")
	UserAssetService := services.NewUserAssetService(authCache, s.Config, &testUserAssetRepository, &dto.ExternalServicesRequestErr{})
	createdAsset, err := UserAssetService.CreateAsset(denominations, userId)

	userAsset, err := UserAssetService.GetAssetById(createdAsset[0].ID)

	assert.Equal(s.T(), nil, err, "Expected CreateAsset to return error")
	assert.Equal(s.T(), testDenominations[0].AssetSymbol, userAsset.AssetSymbol, "Assets not successfully created")
	assert.Equal(s.T(), "0", userAsset.AvailableBalance, "Assets not successfully created")
	assert.Equal(s.T(), testDenominations[0].Decimal, userAsset.Decimal, "Assets not successfully created")
	assert.Equal(s.T(), testUserId1, userAsset.UserID, "Assets not successfully created")
}

func (s *Suite) Test_GetAssetById_Fails_ForNonExistingUserId() {
	nonExistingUserId, _ := uuid.FromString("a10fce7b-7844-43af-9ed1-e130723a0003")
	UserAssetService := services.NewUserAssetService(authCache, s.Config, &testUserAssetRepository, &dto.ExternalServicesRequestErr{})
	_, err := UserAssetService.GetAssetById(nonExistingUserId)

	assert.NotEqual(s.T(), nil, err, "Expected FetchAssets to return error")
	assert.Equal(s.T(), 400, err.(appError.Err).ErrCode, "Expected FetchAssets to return error")
	assert.Equal(s.T(), "RECORD_NOT_FOUND", err.(appError.Err).ErrType, "Expected FetchAssets to return RECORD_NOT_FOUND")
}

// func (s *Suite) Test_GetAssetByAddress_pass_ForV2Address() {
// 	denominations := []string{"LINK", "ETH", "BNB"}
// 	userId, _ := uuid.FromString("a10fce7b-7844-43af-9ed1-e130723a1ea3")
// 	UserAssetService := services.NewUserAssetService(authCache, s.Config)
// 	createdAsset, err := UserAssetService.CreateAsset(&testUserAssetRepository, denominations, userId)

// 	asset, err := UserAssetService.GetAssetByAddressSymbolAndMemo(&testUserAssetRepository, "bnb10f7jqrvg3d978cgtsqydtlk20y992yeapjzd3a", "BNB", "639469678")
// 	expected := testUserAssets1[0]

// 	assert.Equal(s.T(), nil, err, "Expected GetAssetByAddressSymbolAndMemo to not error")
// 	assert.Equal(s.T(), expected, asset, "Expected asset not returned")
// }
