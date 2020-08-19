package utility

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
