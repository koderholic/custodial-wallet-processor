package test

import (
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"

	"github.com/stretchr/testify/require"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func (s *Suite) Test_HotWalletCreation() {
	supportedAssets := []model.Denomination{}
	hotWallet := []model.HotWalletAsset{}

	if err := services.InitHotWallet(authCache, s.DB, s.Logger, s.Config); err != nil {
		require.NoError(s.T(), err)
	}

	if err := s.DB.Find(&supportedAssets).Error; err != nil {
		require.NoError(s.T(), err)
	}

	if err := s.DB.Find(&hotWallet).Error; err != nil {
		require.NoError(s.T(), err)
	}

	if len(supportedAssets) != len(hotWallet) {
		s.T().Errorf("Expected %d hot wallet accounts to be created, got %d", len(supportedAssets), len(hotWallet))
	}
}
func (s *Suite) Test_BUSDHotWalletCreation() {

	hotWallet := model.HotWalletAsset{}

	if err := services.InitHotWallet(authCache, s.DB, s.Logger, s.Config); err != nil {
		require.NoError(s.T(), err)
	}

	if err := s.DB.Where(model.HotWalletAsset{AssetSymbol: "BUSD"}).First(&hotWallet).Error; err != nil {
		if err.Error() != utility.SQL_404 {
			require.NoError(s.T(), err)
		}
		s.T().Errorf("Expected BUSD hot wallet account to be created, got %d", 404)
	}
}
