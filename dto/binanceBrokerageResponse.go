
package dto

type GetUserAddressResponse struct {
	Address string `json:"address"`
	AssetSymbol string `json:"coin"`
	Tag string `json:"tag"`
}