package permissions

var (
	All = map[string]string{
		"GetUserAssets":      "get-assets",
		"CreateUserAssets":   "create-assets",
		"CreditUserAsset":    "credit-asset",
		"DebitUserAsset":     "debit-asset",
		"InternalTransfer":   "do-internal-transfer",
		"GetAssetAddress":    "get-address",
		"GetTransaction":     "get-transactions",
		"OnChainDeposit":     "on-chain-deposit",
		"ConfirmTransaction": "confirm-transaction",
		"ExternalTransfer":   "do-external-transfer",
		"TriggerFloat":       "trigger-float-management",
	}
)
