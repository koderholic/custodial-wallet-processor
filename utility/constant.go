package utility

const (
	SUCCESS                   = "Request Proccessed Successfully"
	INPUT_ERR                 = "Invalid Input Supplied. See documentation"
	SYSTEM_ERR                = "Request Could Not Be Proccessed. Server encountered an error"
	VALIDATION_ERR            = "Validation Failed For Some Fields"
	UUID_CAST_ERR             = "Cannot cast Id, ensure to be passing a valid id"
	EMPTY_AUTH_KEY            = "Authentication token is required"
	INVALID_AUTH_TOKEN        = "Authentication token is not valid"
	AUTH_VALIDATE_ERR         = "Failed to validate authentication token"
	INVALID_TOKENTYPE         = "Access forbidden for non-service token type"
	INVALID_PERMISSIONS       = "Access forbidden, appropriate permission not granted"
	UNKNOWN_ISSUER            = "Access forbidden for unknown token issuer"
	NON_MATCHING_DENOMINATION = "Non matching asset denomination, ensure initiator and recipient has same denomination"
	TRANSFER_TO_SELF          = "Transfer to self not allowed"
	INSUFFICIENT_FUNDS        = "User asset do not have sufficient balance for this transaction"
	SQL_404                   = "record not found"
	INVALID_DEBIT             = "Debit reference provided was not successful."
	INVALID_DEBIT_AMOUNT      = "Value in debit reference does not match value provided."
	DEBIT_PROCESSED_ERR       = "Debit reference has already been processed for external transfer"
	TIMEOUT_ERR               = "Request timedout, taking longer than usual to process"
	MINIMUM_SPENDABLE_ERR     = "Transfer amount is lower than the minimum allowed"
	SVCS_CRYPTOADAPTER_ERR    = "SVCS_CRYPTOADAPTER_ERR"
	SVCS_KEYMGT_ERR           = "SVCS_KEYMGT_ERR"
	NO_MEMO                   = "NO MEMO"
	WITHDRAWALPROCESS         = "WITHDRAW"
	FLOATPROCESS              = "FLOAT"
	SWEEPPROCESS              = "SWEEP"
	SWEEPMEMOBNB              = "ADCXADF1829038FGX"
	BNBTOKENSLIP              = 714
)

var (
	MINIMUM_SPENDABLE = map[string]int64{
		"BTC": 546,
		"ETH": 15000000000000,
		"BNB": 37500,
	}
)
