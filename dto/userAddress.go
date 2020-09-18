package dto

type AssetAddress struct {
	Address string `json:"address,omitempty"`
	Memo    string `json:"memo,omitempty"`
	Type    string `json:"type,omitempty"`
}

type AllAssetAddresses struct {
	Addresses          []AssetAddress `json:"addresses"`
	DefaultAddressType string         `json:"defaultAddressType"`
}
