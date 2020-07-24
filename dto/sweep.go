package dto

import "math/big"

// BTCSweepParam ... Model definition for BTC sweep
type BTCSweepParam struct {
	FloatAddress     string
	BrokerageAddress string
	FloatPercent     *big.Int
	BrokeragePercent *big.Int
}
