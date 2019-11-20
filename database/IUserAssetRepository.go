package database

type IUserAssetRepository interface {
	IRepository
	GetTotalUnconfirmedAccounts(userId int) int
}

type UserAssetRepository struct {
	BaseRepository
}

func (u *UserAssetRepository) GetTotalUnconfirmedAccounts(userId int) int {
	return 2334
}
