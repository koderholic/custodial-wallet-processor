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
	case "subscribeAddressV1":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.CryptoAdapterService,
			Action:   "/webhook/register",
		}
	case "subscribeAddressV2":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.CryptoAdapterService,
			Action:   "/subscription/register",
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
	case "sendEmail":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.NotificationServiceUrl,
			Action:   "/emails/send",
		}
	case "sendSms":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.NotificationServiceUrl,
			Action:   "/sms/send",
		}
	case "createAllAddresses":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.KeyManagementService,
			Action:   "/address/create-all-versions",
		}
	case "getAssetDenominations":
		return MetaData{
			Type:     http.MethodGet,
			Endpoint: config.RateServiceUrl,
			Action:   "/denominations",
		}
	case "getTWDenominations":
		return MetaData{
			Type:     http.MethodGet,
			Endpoint: config.TWServiceUrl,
			Action:   "/wallet-core/master/coins.json",
		}
	case "getUserAddressBBS":
		return MetaData{
			Type:     http.MethodGet,
			Endpoint: config.BinanceBrokerageServiceURL,
			Action:   "/users",
		}
	case "sweepUserAddress":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.BinanceBrokerageServiceURL,
			Action:   "/users",
		}
	case "sendSingleTransaction":
		return MetaData{
		Type:     http.MethodPost,
		Endpoint: config.TransactionSignersURL,
		Action:   "/transactions/send-single",
		}
	case "sendBatchTransaction":
		return MetaData{
			Type:     http.MethodPost,
			Endpoint: config.TransactionSignersURL,
			Action:   "/transactions/send-batch",
		}

	default:
		return MetaData{}
	}
}
