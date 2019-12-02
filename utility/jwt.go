package utility

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	Config "wallet-adapter/config"

	jwt "github.com/dgrijalva/jwt-go"
)

// VerifyJWT ... This verrifies a JWT generated token
func VerifyJWT(authToken string, config Config.Data, tokenClaims interface{}) error {

	authenticatorKey := config.AuthenticatorKey
	keyByte, err := base64.URLEncoding.DecodeString(authenticatorKey)
	if err != nil {
		return err
	}

	token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		rsa, err := jwt.ParseRSAPublicKeyFromPEM(keyByte)
		if err != nil {
			return nil, err
		}
		return rsa, nil
	})

	if err != nil {
		return err
	}

	jwtClaims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		base64Bytes, err := json.Marshal(jwtClaims)
		if err != nil {
			return err
		}
		json.Unmarshal(base64Bytes, tokenClaims)
		return nil
	}

	return errors.New("Failed to validate token")
}
