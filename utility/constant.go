package utility

const (
	NO_MEMO                             = "NO MEMO"
	WITHDRAWALPROCESS                   = "WITHDRAW"
	FLOATPROCESS                        = "FLOAT"
	SWEEPPROCESS                        = "SWEEP"
	SWEEPMEMOBNB                        = "9999999999999"
	BNBTOKENSLIP                        = 714
	ADDRESS_VERSION_V1                  = "VERSION_1"
	ADDRESS_VERSION_V2                  = "VERSION_2"
	SHARED_ADDRESS_ID                   = "56234b22-6f1b-4e47-b9bf-feaa68c0ae99"
	EMPTY_MEMO_ERR                      = "Memo is required"
	COIN_BTC                            = "BTC"
	COIN_BNB                            = "BNB"
	COIN_BUSD                           = "BUSD"
	COIN_ETH                            = "ETH"
	FAILED                              = "FAILED"
	SUCCESSFUL                          = "SUCCESS"
	ADDRESS_TYPE_LEGACY                 = "Legacy"
	ADDRESS_TYPE_SEGWIT                 = "Segwit"
	BTC_COINTYPE                        = 0
	ETH_COINTYPE                        = 60
	BNB_COINTYPE                        = 714
	BNB_COLD_WALLET_ADDRESS             = "bnb136ns6lfw4zs5hg4n85vdthaad7hq5m4gtkgf23"
	BNB_COLD_WALLET_ADDRESS_MEMO        = 109630239
	BTC_COLD_WALLET_ADDRESS             = "17mN6BpX8k7TecMGvC1jWqztd8SYF7VbpZ"
	ETH_COLD_WALLET_ADDRESS             = "0xad7651a207ab7a0fdcefc30c5a4fcc68d830b2f5"
	ENV_PRODUCTION                      = "production"
	ENV_STAGING                         = "staging"
	NOTIFICATION_SMS_TYPE               = "OTHERS"
	NOTIFICATION_SMS_COUNTRY            = "NG"
	INSUFFICIENT_BALANCE_FLOAT_SEND_SMS = "INSUFFICIENT_BALANCE_FLOAT_SEND_SMS"
	ONE_HOUR_MILLISECONDS               = 3600000
	SIX_HUNDRED_MILLISECONDS            = 600000
	SEPERATOR                           = "_"
)

var (
	MINIMUM_SPENDABLE = map[string]float64{
		"BTC": 0.00000546,
		"ETH": 0.000015,
		"BNB": 0.000375,
	}
	DefaultAddressesTypes = map[int64]string{
		BTC_COINTYPE: ADDRESS_TYPE_SEGWIT,
		ETH_COINTYPE: "",
		BNB_COINTYPE: "",
	}
)
