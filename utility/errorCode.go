package utility

var (
	SUCCESS                         = "WAS000"
	INPUTERROR                      = "WAS888"
	SYSTEMERROR                     = "WAS999"
	VALIDATIONERR                   = "WAS-V999"
	INVALIDUSER                     = "WAS-USR001"
)

func GetCodeMsg(errCode string) string {
	switch errCode {
	case "WAS000":
		return "Request Proccessed Successfully"
	case "WAS999":
		return "Request Could Not Be Proccessed. Server encountered an error"
	case "WAS888":
		return "Invalid Input Supplied"
	case "WAS-V999":
		return "Validation Failed For Some Fields"
	case "WAS-USR001":
		return "User Does Not Exist"
	default:
		return "Request Is Being Processed"
	}
}
