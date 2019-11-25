package utility

var (
	SUCCESS       = "WAS-200"
	INPUTERROR    = "WAS-400"
	SYSTEMERROR   = "WAS-500"
	VALIDATIONERR = "WAS-V400"
	INVALIDUSER   = "WAS-USR404"
	UUIDCASTERROR = "WAS-UUIDCAST400"
)

// GetCodeMsg ... Receives an error code and returns the appropriate error string
func GetCodeMsg(errCode string) string {
	switch errCode {
	case "WAS-200":
		return "Request Proccessed Successfully"
	case "WAS-500":
		return "Request Could Not Be Proccessed. Server encountered an error"
	case "WAS-400":
		return "Invalid Input Supplied"
	case "WAS-V400":
		return "Validation Failed For Some Fields"
	case "WAS-USR404":
		return "User Does Not Exist"
	case "WAS-UUIDCAST400":
		return "Cannot cast Id"
	default:
		return "Request Is Being Processed"
	}
}
