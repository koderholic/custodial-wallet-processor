package validator

import (
	"errors"
	"net/http"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/errorcode"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	validation "gopkg.in/go-playground/validator.v9"
	en_translations "gopkg.in/go-playground/validator.v9/translations/en"
)

// CustomizeValidationMessages ... Customize validation error messages
func CustomizeMessages(validator *validation.Validate) (ut.Translator, error) {
	translator := en.New()
	uni := ut.New(translator, translator)

	trans, found := uni.GetTranslator("en")
	if !found {
		return trans, appError.Err{ErrType: errorcode.SERVER_ERR_CODE, ErrCode: http.StatusInternalServerError, Err: errors.New("translator not found")}
	}

	if err := en_translations.RegisterDefaultTranslations(validator, trans); err != nil {
		return trans, appError.Err{ErrType: errorcode.SERVER_ERR_CODE, ErrCode: http.StatusInternalServerError, Err: err}
	}

	_ = validator.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} is a required field", true)
	}, func(ut ut.Translator, fe validation.FieldError) string {
		t, _ := ut.T("required", fe.Field())
		return t
	})

	_ = validator.RegisterTranslation("email", trans, func(ut ut.Translator) error {
		return ut.Add("email", "{0} must be a valid email", true)
	}, func(ut ut.Translator, fe validation.FieldError) string {
		t, _ := ut.T("email", fe.Field())
		return t
	})

	return trans, nil
}
