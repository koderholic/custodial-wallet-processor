package test

import (
	"wallet-adapter/dto"
	"wallet-adapter/services"
	"wallet-adapter/utility"

	"github.com/stretchr/testify/require"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func (s *Suite) Test_HotWalletCreation() {
	supportedAssets := []dto.Denomination{}
	hotWallet := []dto.HotWalletAsset{}

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

	hotWallet := dto.HotWalletAsset{}

	if err := services.InitHotWallet(authCache, s.DB, s.Logger, s.Config); err != nil {
		require.NoError(s.T(), err)
	}

	if err := s.DB.Where(dto.HotWalletAsset{AssetSymbol: "BUSD"}).First(&hotWallet).Error; err != nil {
		if err.Error() != utility.SQL_404 {
			require.NoError(s.T(), err)
		}
		s.T().Errorf("Expected BUSD hot wallet account to be created, got %d", 404)
	}
}
