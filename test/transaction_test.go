package test

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"
	config "wallet-adapter/config"
	"wallet-adapter/controllers"
	"wallet-adapter/database"
	"wallet-adapter/middlewares"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/DATA-DOG/go-sqlmock"

	httpSwagger "github.com/swaggo/http-swagger"
	validation "gopkg.in/go-playground/validator.v9"
)

//TestingSuite ...
type TestingSuite struct {
	suite.Suite
	DB       *gorm.DB
	Mock     sqlmock.Sqlmock
	Database database.Database
	Logger   *utility.Logger
	Config   config.Data
	Router   *mux.Router
}

func TesInitialize(t *testing.T) {
	suite.Run(t, new(TestingSuite))
}

// SetupSuite ...
func (s *TestingSuite) SetupSuite() {

	var (
		db  *sql.DB
		err error
	)

	db, s.Mock, err = sqlmock.New()
	require.NoError(s.T(), err)
	s.DB, err = gorm.Open("mysql", db)
	require.NoError(s.T(), err)
	s.DB.LogMode(true)

	logger := utility.NewLogger()
	router := mux.NewRouter()
	Config := config.Data{
		AppPort:            "9000",
		ServiceName:        "crypto-wallet-adapter",
		AuthenticatorKey:   "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
		PurgeCacheInterval: 5,
	}

	Database := database.Database{
		Logger: logger,
		Config: Config,
		DB:     s.DB,
	}

	s.Database = Database
	s.Logger = logger
	s.Config = Config
	s.Router = router

	s.RegisterRoutes(s.Logger, s.Config, s.Router, validation.New())
}

// RegisterRoutes ...
func (s *TestingSuite) RegisterRoutes(logger *utility.Logger, Config config.Data, router *mux.Router, validator *validation.Validate) {

	once.Do(func() {

		baseRepository := database.BaseRepository{Database: s.Database}
		controller := controllers.NewController(s.Logger, s.Config, validator, &baseRepository)

		apiRouter := router.PathPrefix("").Subrouter()
		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// User Asset Routes
		apiRouter.HandleFunc("/assets/transactions/{reference}", middlewares.NewMiddleware(logger, Config, controller.GetTransaction).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/transactions", middlewares.NewMiddleware(logger, Config, controller.GetTransactionsByAssetId).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Build()).Methods(http.MethodGet)

	})
}

func (s *TestingSuite) Test_GetTransaction() {
	s.Mock.ExpectQuery(regexp.QuoteMeta(
		fmt.Sprintf("SELECT * FROM `transactions`"))).
		WithArgs("9b7227pba3d915ef756a").
		WillReturnRows(sqlmock.NewRows([]string{"id", "initiator_id", "recipient_id", "value", "transaction_status", "transaction_reference", "payment_reference", "previous_balance", "available_balance", "transaction_type", "transaction_end_date", "transaction_start_date", "created_at", "updated_at", "deleted_at", "transaction_tag"}).AddRow("2553225e-5ca8-4688-b0f8-caac2a57d67d", "7118292e-1859-4138-85cc-f8634a1bd196", "7118292e-1859-4138-85cc-f8634a1bd196", 46.090000000000000000, "Completed", "9b7227pba3d915ef756a", "368Z7C92QPRAYNI5", 52.180000000000010000, 6.090000000000010000, "Offchain", time.Now(), time.Now(), time.Now(), time.Now(), time.Now(), "Transfer"))

	getTransactionRequest, _ := http.NewRequest("GET", test.GetTransactionByRef, bytes.NewBuffer([]byte("")))
	getTransactionRequest.Header.Set("x-auth-token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJTVkNTL0FVVEgiLCJwZXJtaXNzaW9ucyI6WyJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5nZXQtdHJhbnNhY3Rpb25zIl0sInRva2VuVHlwZSI6IlNFUlZJQ0UifQ.bDXBFdRYHFHmwksv7_IOxFZWtp6oNsgd1GFn1p1DWxeSK7P6XqJMKU8OwDwpCv189IE1QsUmYmPFasqfK2yxfzx9OTtPRY_7dvUfWwACwJsy0pGd4s8hc1hSmsQHrvSrx-f9Ca2McgZEPWgHSZW1bMbqvonUOJdWcrcICDpA5bwWCnZ_RGD8hKlMV_oKeANYqWE--Zl8r4u_vEM1yPPzANm8cXNBX-cKxeyB-yPbK0mFTKC-_ptC-hADG26bwmtG3Mdv26uViX7q2bc993uhyRuOSRMEzvCIz_-oO5feoiL5KczLvMj8DGMZ_AVPGRjuY3QmVW0yyG2lE5iz0TOC-Q")

	getTransactionResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(getTransactionResponse, getTransactionRequest)

	if getTransactionResponse.Code != http.StatusOK {
		s.T().Errorf("Expected response code to not be %d. Got %d\n", http.StatusOK, getTransactionResponse.Code)
	}
}
