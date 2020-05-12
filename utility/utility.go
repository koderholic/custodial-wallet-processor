package utility

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"math/rand"
	"strconv"
	"time"
)

func RandNo(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
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
