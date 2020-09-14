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
	denomination := model.Denomination{
		Decimal: 8,
	}
	result := tasks.ConvertBigIntToDecimalUnit(*amount, denomination)
	if !strings.EqualFold(fmt.Sprintf("%f", result), "0.000017") {
		t.Fail()
	}
}
