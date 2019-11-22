package model

type ChainTransaction struct {
	BaseModel
	Status      bool   `gorm:"index;not null;default:false" json:"status"`
	Hash        string `json:"hash"`
	BlockHeight int64  `json:"blockHeight"`
}
