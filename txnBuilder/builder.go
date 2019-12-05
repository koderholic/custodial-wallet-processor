package coin

type ITxBuilder interface {
	GenerateAddress(seedWords string, nextDerivationPath string) string
	GenerateDerivationPath(lastDerivationPath int64) string
}

type BaseCoin struct {
}

//TODO accept coin name and return coin struct
func NewCoin(coinSymbol string) ICoin {
	var coin ICoin
	switch coinSymbol {
	case "BTC":
		coin = &Bitcoin{}
	case "ETH":
		coin = &Ethereum{}
	case "BNB":
		coin = &Binance{}
	default:
		coin = &Bitcoin{}
	}

	return coin
}
