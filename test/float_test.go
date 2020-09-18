package test

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"
	"wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/model"
	"wallet-adapter/tasks"
	"wallet-adapter/utility"
)

func (s *Suite) TestFloat() {
	configTest := config.Data{
		AppPort:                   "9000",
		ServiceName:               "crypto-wallet-adapter",
		AuthenticatorKey:          "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
		PurgeCacheInterval:        5,
		ExpireCacheDuration:       400,
		ServiceID:                 "4b0bde7a-9201-4cf9-859f-e61d976e376d",
		ServiceKey:                "32e1f6396de342e879ca07ec68d4d907",
		AuthenticationService:     "https://internal.dev.bundlewallet.com/authentication",
		KeyManagementService:      "https://internal.dev.bundlewallet.com/key-management",
		CryptoAdapterService:      "https://internal.dev.bundlewallet.com/crypto-adapter",
		DepositWebhookURL:         "http://internal.dev.bundlewallet.com/crypto-adapter/incoming-deposit",
		BtcSlipValue:              "0",
		LockerService:             "https://internal.dev.bundlewallet.com/locker",
		FloatCronInterval:         "1/5 * * * *",
		WithdrawToHotWalletUrl:    "http://internal.dev.bundlewallet.com/order-book",
		NotificationServiceUrl:    "http://internal.dev.bundlewallet.com/notifications",
		ColdWalletEmail:           "akinyemi@bundle.africa",
		ColdWalletEmailTemplateId: "to-be-set",
	}

	type SupportedAsset struct {
		MinimumSweep float64 `mapstructure:"minimumSweep"  yaml:"minimumSweep,omitempty"`
	}

	purgeInterval := configTest.PurgeCacheInterval * time.Second
	cacheDuration := configTest.ExpireCacheDuration * time.Second
	authCache := utility.InitializeCache(cacheDuration, purgeInterval)
	baseRepository := database.BaseRepository{Database: s.Database}
	userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}
	tasks.ManageFloat(authCache, configTest, baseRepository, userAssetRepository)

}

func TestConversion(t *testing.T) {
	amount := big.NewInt(1699)
	denomination := model.Denomination{
		Decimal: 8,
	}
	result := tasks.ConvertBigIntToDecimalUnit(*amount, denomination)
	fmt.Println(fmt.Sprintf("%s %f", "BUSD", result))
	if !strings.EqualFold(fmt.Sprintf("%f", result), "0.000017") {
		t.Fail()
	}
}
