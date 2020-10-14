package denoms

var (
	MinimumSpendable = map[string]float64{
		"BTC": 0.00000546,
		"ETH": 0.000015,
		"BNB": 0.000375,
	}
	AddressTypes = map[int64][]string{
		0: {"Segwit", "Legacy"},
	}
)

const (
	BNB_COLD_WALLET_ADDRESS      = "bnb136ns6lfw4zs5hg4n85vdthaad7hq5m4gtkgf23"
	BNB_COLD_WALLET_ADDRESS_MEMO = 109630239
	BTC_COLD_WALLET_ADDRESS      = "17mN6BpX8k7TecMGvC1jWqztd8SYF7VbpZ"
	ETH_COLD_WALLET_ADDRESS      = "0xad7651a207ab7a0fdcefc30c5a4fcc68d830b2f5"
	COIN_BTC                     = "BTC"
	COIN_BNB                     = "BNB"
	COIN_BUSD                    = "BUSD"
	COIN_ETH                     = "ETH"
	BTC_COINTYPE                 = 0
	ETH_COINTYPE                 = 60
	BNB_COINTYPE                 = 714
)
