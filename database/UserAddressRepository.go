package database

// ITransactionRepository ...
type IUserAddressRepository interface {
	IUserAssetRepository
}

// UserAddressRepository ...
type UserAddressRepository struct {
	UserAssetRepository
}
