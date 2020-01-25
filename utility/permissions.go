package utility

var (
	Permissions = map[string]string{
		"GetUserAssets":         "get-assets",
		"CreateUserAssets":      "create-assets",
		"CreditUserAsset":       "credit-asset",
		"DebitUserAsset":        "debit-asset",
		"InternalTransfer":      "do-internal-transfer",
		"GetAssetAddress":       "get-address",
		"GetUserAssetById":      "get-asset-byid",
		"GetUserAssetByAddress": "get-asset-byaddr",
	}
)
