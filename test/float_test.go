package test

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
	"wallet-adapter/model"
	"wallet-adapter/tasks"
)

func TestConversion(t *testing.T) {
	amount := big.NewInt(1699)
	network := model.Network{
		NativeDecimals: 8,
	}
	result := tasks.ConvertBigIntToDecimalUnit(*amount, network)
	if !strings.EqualFold(fmt.Sprintf("%f", result), "0.000017") {
		t.Fail()
	}
}
