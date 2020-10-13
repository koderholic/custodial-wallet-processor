package jwt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	Config "wallet-adapter/config"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/errorcode"

	jwt "github.com/dgrijalva/jwt-go"
)

var (
	X_AUTH_TOKEN = "x-auth-token"
)

// VerifyJWT ... This verrifies a JWT generated token
func Verify(authToken string, config Config.Data, tokenClaims interface{}) error {

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

// DecodeToken ... This decodes a JWT generated token
func DecodeToken(authToken string, config Config.Data, tokenClaims interface{}) error {

	authenticatorKey := config.AuthenticatorKey
	keyByte, err := base64.URLEncoding.DecodeString(authenticatorKey)
	if err != nil {
		return appError.Err{ErrType: errorcode.INPUT_ERR_CODE, ErrCode: http.StatusInternalServerError, Err: err}
	}

	token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, appError.Err{ErrType: errorcode.SERVER_ERR_CODE, ErrCode: http.StatusInternalServerError,
				Err: errors.New(fmt.Sprintf("Unexpected signing method: %v", token.Header["alg"]))}
		}
		rsa, err := jwt.ParseRSAPublicKeyFromPEM(keyByte)
		if err != nil {
			return nil, appError.Err{ErrType: errorcode.SERVER_ERR_CODE, ErrCode: http.StatusInternalServerError, Err: err}
		}
		return rsa, nil
	})

	if err != nil {
		return err
	}

	jwtClaims, _ := token.Claims.(jwt.MapClaims)
	base64Bytes, err := json.Marshal(jwtClaims)
	if err != nil {
		return appError.Err{ErrType: errorcode.SERVER_ERR_CODE, ErrCode: http.StatusInternalServerError, Err: err}
	}
	json.Unmarshal(base64Bytes, tokenClaims)
	return nil
}
