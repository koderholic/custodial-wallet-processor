package utility

var (
	Permissions = map[string]string{
		"GetUserAssets":    "get-assets",
		"CreateUserAssets": "create-assets",
		"CreditUserAsset":  "credit-asset",
		"DebitUserAsset":   "debit-asset",
		"InternalTransfer": "do-internal-transfer",
	}
)
