package utility

import (
	"gopkg.in/go-playground/validator.v9"
)

// ResponseObj... Response object definition without additional data field
type ResponseObj struct {
	Status  bool
	Code    string
	Message string
}

// ResponseResultObj... Response object definition with additional data field
type ResponseResultObj struct {
	ResponseObj
	Data interface{}
}

// ResponseResultObj... Validation error object
type ResponseValidateObj struct {
	ResponseObj
	ValidationErrors validator.ValidationErrors
}

// NewResponse ... Initializes a response object.
func NewResponse() ResponseResultObj {
	return ResponseResultObj{}
}

// PlainSuccess ... Returns successful response without additional data
func (res ResponseResultObj) PlainSuccess(code string, msg string) ResponseObj {

	response := ResponseObj{}
	response.Status = true
	response.Code = code
	response.Message = msg

	return response
}

// Success ... Returns successful response with additional data
func (res ResponseResultObj) Success(code string, msg string, data interface{}) ResponseResultObj {
	res.Status = true
	res.Code = code
	res.Message = msg
	res.Data = data
	return res
}

// PlainError ... Returns error response with no additional data
func (res ResponseResultObj) PlainError(code string, err string) ResponseObj {
	return ResponseObj{
		Status:  false,
		Code:    code,
		Message: err,
	}
}

// Error ... Returns error response with additional data
func (res ResponseResultObj) Error(code string, err string, data interface{}) ResponseResultObj {
	res.Status = true
	res.Code = code
	res.Message = err
	res.Data = data
	return res
}

// ValidateError... Returns error response with validation messages
func (res ResponseResultObj) ValidateError(code string, err string, errors validator.ValidationErrors) ResponseValidateObj {
	response := ResponseValidateObj{}
	response.Status = false
	response.Code = code
	response.Message = err
	response.ValidationErrors = errors
	return response
}
