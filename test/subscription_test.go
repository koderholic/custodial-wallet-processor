package test

import (
	"fmt"
	"testing"
	"wallet-adapter/config"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"
)

func TestSubscribeAddress(t *testing.T) {

	logger := utility.NewLogger()
	supportedCoins := make(map[string]config.SupportedCoin)
	supportedCoins["bitcoin"] = config.SupportedCoin{
		Name:   "bitcoin",
		Symbol: "btc",
		Slip:   "0",
	}
	configTest := config.Data{
		AppPort:               "9000",
		ServiceName:           "crypto-wallet-adapter",
		AuthenticatorKey:      "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
		PurgeCacheInterval:    5,
		ServiceID:             "4b0bde7a-9201-4cf9-859f-e61d976e376d",
		ServiceKey:            "32e1f6396de342e879ca07ec68d4d907",
		AuthenticationService: "https://internal.dev.bundlewallet.com/authentication",
		KeyManagementService:  "https://internal.dev.bundlewallet.com/key-management",
		CryptoAdapterService:  "https://internal.dev.bundlewallet.com/crypto-adapter",
		DepositWebhookURL:     "http://internal.dev.bundlewallet.com/crypto-adapter/incoming-deposit",
		SupportedCoins:        supportedCoins,
	}

	subs := make(map[string][]string)
	addressArray := []string{"bc1qq65rn0wzjnmcwqz4jp0cx48lvj7ynectmw60jj"}
	subs[configTest.SupportedCoins["bitcoin"].Slip] = addressArray
	fmt.Print(subs["0"])
	requestData := model.SubscriptionRequest{
		Subscriptions: subs,
		Webhook:       configTest.DepositWebhookURL,
	}
	subscriptionResponseData := model.SubscriptionResponse{}
	serviceErr := model.ServicesRequestErr{}

	services.SubscribeAddress(logger, configTest, requestData, &subscriptionResponseData, serviceErr)

}
