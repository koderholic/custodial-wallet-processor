package utility

import (
	"encoding/json"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

func RandNo(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min

}

func NativeValue(denominationDecimal int, rawValue decimal.Decimal) decimal.Decimal {
	conversionDecimal := decimal.NewFromInt(int64(denominationDecimal))
	baseExp := decimal.NewFromInt(10)
	return rawValue.Mul(baseExp.Pow(conversionDecimal))
}

//GenerateReferenceID ....
func RandomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// UnmarshalJsonFile ... This handles reading from file and writing into a receiver object
func UnmarshalJsonFile(fileLocation string, contentReciever interface{}) error {
	jsonBytes, err := ioutil.ReadFile(fileLocation)
	if err != nil {
		println(err.Error)
		return err
	}
	err = json.Unmarshal([]byte(jsonBytes), contentReciever)
	if err != nil {
		println(err.Error)
		return err
	}
	return nil
}

// The special precision -1 uses the smallest number of digits
// necessary such that ParseFloat will return f exactly
const DigPrecision = -1

func Add(value float64, availableBalance string, decimals int) string {
	availBal, _ := strconv.ParseFloat(availableBalance, 64)
	currentAvailableBalance := availBal*math.Pow10(decimals) + value*math.Pow10(decimals)
	currentAvailableBalanceString := strconv.FormatFloat(currentAvailableBalance/math.Pow10(decimals), 'g', DigPrecision, 64)
	return currentAvailableBalanceString
}

func Subtract(value float64, availableBalance string, decimals int) string {
	availBal, _ := strconv.ParseFloat(availableBalance, 64)
	currentAvailableBalance := availBal*math.Pow10(decimals) - value*math.Pow10(decimals)
	currentAvailableBalanceString := strconv.FormatFloat(currentAvailableBalance/math.Pow10(decimals), 'g', DigPrecision, 64)
	return currentAvailableBalanceString
}

func IsGreater(value float64, availableBalance string, decimals int) bool {
	availBal, _ := strconv.ParseFloat(availableBalance, 64)
	if availBal*math.Pow10(decimals)-value*math.Pow10(decimals) < 0 {
		return false
	}
	return true
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func GetSingleTXProcessingIntervalTime(n int) int {
	SLEEP_INTERVAL := n * 5
	SLEEP_INTERVAL = MinInt(SLEEP_INTERVAL, 20)
	return SLEEP_INTERVAL
}

func MaxFloat(a, b *big.Float) *big.Float {
	if a.Cmp(b) >= 0 {
		return a
	}
	return b
}

func MinFloat(a, b *big.Float) *big.Float {
	if a.Cmp(b) <= 0 {
		return a
	}
	return b
}

func IsExceedWaitTime(startTime, endTime time.Time) bool {
	if startTime.After(endTime) {
		return true
	}
	return false
}

func FloatToString(input_num float64) string {
	return strconv.FormatFloat(input_num, 'f', 8, 64)
}

func IsValidUUID(u string) bool {
	_, err := uuid.FromString(u)
	return err == nil
}

func GetNextDayFromNow() *time.Time {
	nextDayFromNow := time.Now().Add(time.Duration(24 - time.Now().Hour())* time.Hour)
 return &nextDayFromNow
}