package response

// ResponseObj ... Response object definition without additional data field
type ResponseObj struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ResponseResultObj ... Response object definition with additional data field
type ResponseResultObj struct {
	ResponseObj
	Data interface{} `json:"data"`
}

// ResponseResultObj ... Validation error object
type ResponseValidateObj struct {
	ResponseObj
	ValidationErrors []map[string]string
}

// New ... Initializes a response object.
func New() ResponseResultObj {
	return ResponseResultObj{}
}

// PlainSuccess ... Returns successful response without additional data
func (res ResponseResultObj) PlainSuccess(code string, msg string) ResponseObj {

	response := ResponseObj{}
	response.Success = true
	response.Code = code
	response.Message = msg

	return response
}

// Success ... Returns successful response with additional data
func (res ResponseResultObj) Successful(code string, msg string, data interface{}) ResponseResultObj {

	response := ResponseObj{}
	response.Success = true

	res.Success = response.Success
	res.Code = code
	res.Message = msg
	res.Data = data
	return res
}

// PlainError ... Returns error response with no additional data
func (res ResponseResultObj) PlainError(code string, err string) ResponseObj {
	return ResponseObj{
		Success: false,
		Code:    code,
		Message: err,
	}
}

// Error ... Returns error response with additional data
func (res ResponseResultObj) Error(code string, err string, data interface{}) ResponseResultObj {
	// res.Success = false
	res.Code = code
	res.Message = err
	res.Data = data
	return res
}

// ValidateError ... Returns error response with validation messages
func (res ResponseResultObj) ValidateError(code string, err string, errors []map[string]string) ResponseValidateObj {
	response := ResponseValidateObj{}
	response.Success = false
	response.Code = code
	response.Message = err
	response.ValidationErrors = errors
	return response
}
