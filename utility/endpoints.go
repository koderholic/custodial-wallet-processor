package utility

import (
	"net/http"
	Config "wallet-adapter/config"
)

type MetaData struct {
	Type, Endpoint, Action string
}

// GetRequestMetaData ...
func GetRequestMetaData(request string, config Config.Data) MetaData {
	switch request {
	case "generateToken":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.AuthenticationService,
			Action:   "/services/token",
		}
	case "createAddress":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.KeyManagementService,
			Action:   "/address/create",
		}
	case "signTransaction":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.KeyManagementService,
			Action:   "/sign-transaction",
		}
	case "signBatchTransaction":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.KeyManagementService,
			Action:   "/sign-batch-transaction",
		}
	case "broadcastTransaction":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.CryptoAdapterService,
			Action:   "/broadcast-transaction",
		}
	case "subscribeAddress":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.CryptoAdapterService,
			Action:   "/webhook/register",
		}
	case "transactionStatus":
		return MetaData{
			Type:     http.MethodGet,
			Endpoint: config.CryptoAdapterService,
			Action:   "/transaction-status",
		}
	case "getOnchainBalance":
		return MetaData{
			Type:     http.MethodGet,
			Endpoint: config.CryptoAdapterService,
			Action:   "/onchain-balance",
		}
	case "acquireLock":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.LockerService,
			Action:   "/locks/acquire",
		}
	case "renewLockLease":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.LockerService,
			Action:   "/locks/renew",
		}
	case "releaseLock":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.LockerService,
			Action:   "/locks/release",
		}
	case "withdrawToHotWallet":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.WithdrawToHotWalletUrl,
			Action:   "/brokerage-wallets/withdrawal",
		}
	case "getAssetBalances":
		return MetaData{
			Type:     http.MethodGet,
			Endpoint: config.WithdrawToHotWalletUrl,
			Action:   "/brokerage-wallets/assets-balance",
		}
	case "getDepositAddress":
		return MetaData{
			Type:     http.MethodGet,
			Endpoint: config.WithdrawToHotWalletUrl,
			Action:   "/brokerage-wallets/get-deposit-address",
		}
	default:
		return MetaData{}
	}
}
