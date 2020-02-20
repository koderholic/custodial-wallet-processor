package test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"wallet-adapter/config"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"
)

const (
	okResponse = `{
		"signedData": "01000000000101f0da3bdfa7649bd82e150df03c1f1afd6192c02204201e1e87143c4897d950f30000000000ffffffff016243000000000000160014c20e807c07ab32d83d06ca9fbec3261a949575720247304402201fdb6a0e54178f17d5f0e942a9c60dd2c67a81fabeba665f46d77b8886f6d0e802202c27d3286b77376ac4a7afed60d6b7d1496b8a0b359adae0309da04a24250dee0121022b963c18cfa779aceb86501ef79b846aa6e0784658cbef6aa62e6bdd91a60ca800000000",
		"fee": 5250
	}`
)


func TestSignTransactionImplementation(t *testing.T) {

	logger := utility.NewLogger()
	Config := config.Data{
		AppPort:               "9000",
		ServiceName:           "crypto-wallet-adapter",
		AuthenticatorKey:      "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
		PurgeCacheInterval:    5,
		ExpireCacheDuration:   400,
		ServiceID:             "4b0bde7a-9201-4cf9-859f-e61d976e376d",
		ServiceKey:            "32e1f6396de342e879ca07ec68d4d907",
		AuthenticationService: "https://internal.dev.bundlewallet.com/authentication",
		KeyManagementService:  "https://internal.dev.bundlewallet.com/key-management",
	}

	requestData := model.SignTransactionRequest{
		FromAddress: "0xcDb4D4dbe1a5154E5046b4fBa2efA2FA5E6a64Ec",
		ToAddress:   "0x6CB3F3b958287fD63FA39ED8a392414115c089b3",
		Amount:      1510000000000000,
		CoinType:    "ETH",
	}
	responseData := model.SignTransactionResponse{}
	serviceErr := model.ServicesRequestErr{}

	purgeInterval := Config.PurgeCacheInterval * time.Second
	cacheDuration := Config.ExpireCacheDuration * time.Second
	authCache := utility.InitializeCache(cacheDuration, purgeInterval)

	if err := services.SignTransaction(authCache, logger, Config, requestData, &responseData, serviceErr); err == nil {
		t.Errorf("Expected SignTransaction to error due to validation on request data, got %s\n", err)
	}

	// if responseData.SignedData == "" {
	// 	t.Errorf("Expected SignTransaction to return signed data, got %s\n", responseData.SignedData)
	// }
}

func TestBatchSignBtcImplementation(t *testing.T) {

	logger := utility.NewLogger()
	Config := config.Data{
		AppPort:               "9000",
		ServiceName:           "crypto-wallet-adapter",
		AuthenticatorKey:      "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
		PurgeCacheInterval:    5,
		ExpireCacheDuration:   400,
		ServiceID:             "4b0bde7a-9201-4cf9-859f-e61d976e376d",
		ServiceKey:            "32e1f6396de342e879ca07ec68d4d907",
		AuthenticationService: "https://internal.dev.bundlewallet.com/authentication",
		KeyManagementService:  "https://internal.dev.bundlewallet.com/key-management",
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//assert.Equal(t, "key", r.Header.Get("Key"))
		//assert.Equal(t, "secret", r.Header.Get("Secret"))
		w.Write([]byte(okResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	// Calls key-management to batch sign transaction
	recipientData := []model.BatchRecipients{}
	//get float

	floatRecipient := model.BatchRecipients{
		Address: "bc1qcg8gqlq84veds0gxe20masexr22f2atjn6g6yj",
		Value:   0,
	}
	recipientData = append(recipientData, floatRecipient)
	var btcAssets []string
	btcAssets = append(btcAssets, "bc1qs5gu88wflpnnx5ve9wgay79rn9ajr8masy7akj")

	signTransactionRequest := model.BatchBTCRequest{
		AssetSymbol:   "BTC",
		ChangeAddress: "bc1qcg8gqlq84veds0gxe20masexr22f2atjn6g6yj",
		IsSweep:       true,
		Origins:       btcAssets,
		Recipients:    recipientData,
	}
	signTransactionResponse := model.SignTransactionResponse{}
	serviceErr := model.ServicesRequestErr{}

	purgeInterval := Config.PurgeCacheInterval * time.Second
	cacheDuration := Config.ExpireCacheDuration * time.Second
	authCache := utility.InitializeCache(cacheDuration, purgeInterval)

	if err := services.SignBatchBTCTransaction(httpClient, authCache, logger, Config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
		t.Errorf("Expected SignTransaction to error due to validation on request data, got %s\n", err)
	}

}

func TestBroadcastTransactionImplementation(t *testing.T) {

	logger := utility.NewLogger()
	Config := config.Data{
		AppPort:               "9000",
		ServiceName:           "crypto-wallet-adapter",
		AuthenticatorKey:      "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
		PurgeCacheInterval:    5,
		ServiceID:             "4b0bde7a-9201-4cf9-859f-e61d976e376d",
		ServiceKey:            "32e1f6396de342e879ca07ec68d4d907",
		AuthenticationService: "https://internal.dev.bundlewallet.com/authentication",
		KeyManagementService:  "https://internal.dev.bundlewallet.com/key-management",
		CryptoAdapterService:  "https://internal.dev.bundlewallet.com/crypto-adapter",
	}

	requestData := model.BroadcastToChainRequest{
		SignedData:  "f86a808447868c0082520894c6c55ce8e861119a9013c35e5b93de56b36ee6c0871ff973cafa80008026a09883e0019f6383d22a35aa9ce611717af670cadd5abe44eb2fe8fd2db46cacaca04f2c44b4319ee988e4b78636ec14425b44e945c2d2f583c744f9bf92faadd90c",
		AssetSymbol: "ETH",
	}
	responseData := model.BroadcastToChainResponse{}
	serviceErr := model.ServicesRequestErr{}

	purgeInterval := Config.PurgeCacheInterval * time.Second
	cacheDuration := Config.ExpireCacheDuration * time.Second
	authCache := utility.InitializeCache(cacheDuration, purgeInterval)
	
	if err := services.BroadcastToChain(authCache, logger, Config, requestData, &responseData, serviceErr); err == nil {
		t.Errorf("Expected SignTransaction to error due to incorrect signed data, got %s\n", err)
	}

	// if responseData.TransactionHash == "" {
	// 	t.Errorf("Expected SignTransaction to error due to incorrect signed data, got %s\n", responseData.TransactionHash)
	// }
}

func testingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialTLS: func(network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
		},
	}

	return cli, s.Close
}
