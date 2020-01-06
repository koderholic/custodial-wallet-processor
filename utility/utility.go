package utility

import (
	"math/rand"
	"strconv"
	"time"
)

//GenerateReferenceID ....
func GenerateReferenceID() string {
	currentTime := time.Now()

	return currentTime.Format("060102150405") + GenerateRandomString(12)
}

//GenerateRandomString ....
func GenerateRandomString(length int) string {
	var randomstring = "100"

	for len(randomstring) < length {
		randomstring = randomstring + strconv.Itoa(rand.Intn(9))
	}
	return randomstring
}
