package dto

// ServicesRequestErr ... Model definition for external services request made with error response
type ExternalServicesRequestErr struct {
	Success    bool              `json:"success"`
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	StatusCode int               `json:"_"`
	Data       map[string]string `json:"data"`
}

// ServicesRequestSuccess ... Model definition for external services request made with successful response but no data
type ServicesRequestSuccess struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}

// LockerServiceRequest ... Request definition for  acquire or renew lock, locker service
type LockerServiceRequest struct {
	Identifier   string `json:"identifier"`
	Token        string `json:"token"`
	ExpiresAfter int64  `json:"expiresAfter"`
	Timeout      int64  `json:"timeout"`
}

// LockerServiceResponse ... Model definition for acquire lock successful response, locker service
type LockerServiceResponse struct {
	Identifier string `json:"identifier"`
	Token      string `json:"token"`
	ExpiresAt  string `json:"expiresAt"`
	Fence      int64  `json:"fence"`
}

// LockReleaseRequest ...Request definition for release lock, locker service
type LockReleaseRequest struct {
	Identifier string `json:"identifier"`
	Token      string `json:"token"`
}

// OnchainBalanceRequest ... Request definition for get on-chain balance, crypto-adapter service
type OnchainBalanceRequest struct {
	AssetSymbol string `json:"assetSymbol"`
	Address     string `json:"address"`
}

// OnchainBalanceResponse ... Model definition for get on-chain balance successful response, crypto-adapter service
type OnchainBalanceResponse struct {
	Balance     string `json:"balance"`
	AssetSymbol string `json:"assetSymbol"`
	Decimals    int    `json:"decimals"`
}

type WitdrawToHotWalletRequest struct {
	WithdrawOrderId    string `json:"withdrawOrderId"`
	Network            string `json:"network"`
	Address            string `json:"address"`
	AddressTag         string `json:"addressTag"`
	TransactionFeeFlag bool   `json:"transactionFeeFlag"`
	Name               string `json:"name"`
	Amount             Money  `json:"amount"`
}

type Money struct {
	Value        string `json:"value"`
	Denomination string `json:"denomination"`
}

type WitdrawToHotWalletResponse struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

type BinanceAssetBalances struct {
	CoinList []struct {
		Coin        string `json:"coin"`
		Balance     string `json:"balance"`
		Name        string `json:"name"`
		NetworkList []struct {
			AddressRegex       string `json:"addressRegex"`
			Coin               string `json:"coin"`
			DepositDesc        string `json:"depositDesc"`
			DepositEnable      bool   `json:"depositEnable"`
			IsDefault          bool   `json:"isDefault"`
			MemoRegex          string `json:"memoRegex"`
			MinConfirm         int    `json:"minConfirm"`
			Name               string `json:"name"`
			Network            string `json:"network"`
			ResetAddressStatus bool   `json:"resetAddressStatus"`
			SpecialTips        string `json:"specialTips"`
			UnLockConfirm      int    `json:"unLockConfirm"`
			WithdrawDesc       string `json:"withdrawDesc"`
			WithdrawEnable     bool   `json:"withdrawEnable"`
			WithdrawFee        string `json:"withdrawFee"`
			WithdrawMin        string `json:"withdrawMin"`
		} `json:"networkList"`
	} `json:"coinList"`
}

type DepositAddressResponse struct {
	Address string `json:"address"`
	Coin    string `json:"coin"`
	Tag     string `json:"tag"`
	URL     string `json:"url"`
}

type SendEmailRequest struct {
	Subject   string        `json:"subject"`
	Content   string        `json:"content"`
	Template  EmailTemplate `json:"template"`
	Sender    EmailUser     `json:"sender"`
	Receivers []EmailUser   `json:"receivers"`
	Cc        []EmailUser   `json:"cc"`
	Bcc       []EmailUser   `json:"bcc"`
}

type EmailTemplate struct {
	ID     string            `json:"id"`
	Params map[string]string `json:"params"`
}

type EmailUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type SendEmailResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
		} `json:"data"`
	} `json:"error"`
}

type SendSmsRequest struct {
	Message     string `json:"message"`
	PhoneNumber string `json:"phoneNumber"`
	SmsType     string `json:"smsType"`
	Country     string `json:"country"`
}

type SendSmsResponse struct {
	SendEmailResponse
}
