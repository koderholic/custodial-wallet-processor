package errorcode


const (
	INPUT_ERR                           = "Invalid Input Supplied. See documentation"
	SYSTEM_ERR                          = "Request Could Not Be Proccessed. Server encountered an error"
	VALIDATION_ERR                      = "Validation Failed For Some Fields"
	UUID_CAST_ERR                       = "Cannot cast Id, ensure to be passing a valid id"
	EMPTY_AUTH_KEY                      = "Authentication token is required"
	INVALID_AUTH_TOKEN                  = "Authentication token is not valid"
	AUTH_VALIDATE_ERR                   = "Failed to validate authentication token"
	INVALID_TOKENTYPE                   = "Access forbidden for non-service token type"
	INVALID_PERMISSIONS                 = "Access forbidden, appropriate permission not granted"
	UNKNOWN_ISSUER                      = "Access forbidden for unknown token issuer"
	NON_MATCHING_DENOMINATION           = "Non matching asset denomination, ensure initiator and recipient has same denomination"
	TRANSFER_TO_SELF                    = "Transfer to self not allowed"
	INSUFFICIENT_FUNDS_ERR              = "User asset do not have sufficient balance for this transaction"
	SQL_404                             = "record not found"
	INVALID_DEBIT                       = "Debit reference provided was not successful."
	INVALID_DEBIT_AMOUNT                = "Value in debit reference does not match value provided."
	DEBIT_PROCESSED_ERR                 = "Debit reference has already been processed for external transfer"
	TIMEOUT_ERR                         = "Request timedout, taking longer than usual to process"
	MINIMUM_SPENDABLE_ERR               = "Transfer amount is lower than the minimum allowed"
	EMPTY_MEMO_ERR                      = "Memo is required"
	INSUFFICIENT_BALANCE_FLOAT_SEND_SMS = "INSUFFICIENT_BALANCE_FLOAT_SEND_SMS"
	INSUFFICIENT_FUNDS                  = "INSUFFICIENT_FUNDS"
	BROADCAST_FAILED_ERR                = "TRANSACTION_BROADCAST_FAILED"
	BROADCAST_REJECTED_ERR              = "TRANSACTION_REJECTED_ON_BROADCAST"
	WITHDRAWAL_NOT_ACTIVE               = "Withdrawal operation is currently not available for this asset"
	DEPOSIT_NOT_ACTIVE                  = "deposit operation is currently not available for this asset"
	RECORD_NOT_FOUND                    = "RECORD_NOT_FOUND"
	SERVER_ERR                          = "SERVER_ERR"
	MULTIPLE_ADDRESS_ERROR = "Multiple addresses is not supported for specified asset"
	MULTIPLE_ADDRESS_ERROR_CODE = "MULTIPLE_ADDRESS_NOT_SUPPORTED"
)