package database

// IBatchRepository ...
type IBatchRepository interface {
	IUserAssetRepository
}

// BatchRepository ...
type BatchRepository struct {
	UserAssetRepository
}
