package utility

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"time"
)

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
