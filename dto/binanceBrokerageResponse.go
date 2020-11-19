package dto

type GetUserAddressResponse struct {
	Address     string `json:"address"`
	AssetSymbol string `json:"coin"`
	Tag         string `json:"tag"`
}
type SweepResponse struct {
	ClientTranId string `json:"clientTranId"`
	TxnId        string `json:"txnId"`
}
type SweepRequest struct {
	Amount Money `json:"amount"`
}
