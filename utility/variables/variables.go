package variables

var (
	MINIMUM_SPENDABLE = map[string]float64{
		"BTC": 0.00000546,
		"ETH": 0.000015,
		"BNB": 0.000375,
	}
	AddressTypes = map[int64][]string{
		0: []string{"Segwit", "Legacy"},
	}
)
