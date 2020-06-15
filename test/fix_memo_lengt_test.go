package test

import (
	"testing"
	"wallet-adapter/utility"
	"strconv"

	"github.com/magiconair/properties/assert"
)

func TestFixedMemoGenerationLength(t *testing.T) {
	for i := 0; i <= 100; i++ {
		memo := strconv.Itoa(utility.RandNo(100000000, 999999999))
		assert.Equal(t, len(memo), 9)
	}
}
