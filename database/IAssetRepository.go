package database

type IAssetRepository interface {
	IRepository
	GetTotalUnconfirmedAccounts(userId int) int
}

type AssetRepository struct {
	BaseRepository
}

func (u *AssetRepository) GetTotalUnconfirmedAccounts(userId int) int {
	return 2334
}
