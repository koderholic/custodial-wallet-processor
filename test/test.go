package test

import (
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	config "wallet-adapter/config"
	"wallet-adapter/middlewares"
	"wallet-adapter/utility"

	"github.com/stretchr/testify/require"

	"wallet-adapter/controllers"
	"wallet-adapter/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"
	httpSwagger "github.com/swaggo/http-swagger"
	validation "gopkg.in/go-playground/validator.v9"
)

var (
	once sync.Once
)

type Suite struct {
	suite.Suite
	DB         *gorm.DB
	Mock       sqlmock.Sqlmock
	Database   database.Database
	Logger     *utility.Logger
	Config     config.Data
	Middleware http.Handler
}

func (s *Suite) SetupSuite() {

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
	validator := validation.New()
	Config := config.Data{
		AppPort:            "9000",
		ServiceName:        "wallet-adapter",
		BasePath:           "/api/v1",
		AuthenticatorKey:   "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
		PurgeCacheInterval: 5,
	}

	Database := database.Database{
		Logger: logger,
		Config: Config,
		DB:     s.DB,
	}
	middleware := middlewares.NewMiddleware(logger, Config, router).ValidateAuthToken().LogAPIRequests().Build()

	s.Database = Database
	s.Logger = logger
	s.Config = Config
	s.Middleware = middleware

	s.RegisterRoutes(router, validator)
}

// RegisterRoutes ...
func (s *Suite) RegisterRoutes(router *mux.Router, validator *validation.Validate) {

	once.Do(func() {
		fmt.Printf("s.Config.BasePath >> %+v", s)
		baseRepository := database.BaseRepository{Database: s.Database}
		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}

		// controller := controllers.NewController(s.Logger, s.Config, validator, &baseRepository)
		userAssetController := controllers.NewUserAssetController(s.Logger, s.Config, validator, &userAssetRepository)
		basePath := s.Config.BasePath

		apiRouter := router.PathPrefix(basePath).Subrouter()
		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// User Asset Routes
		apiRouter.HandleFunc("/crypto/users/assets", userAssetController.CreateUserAssets).Methods(http.MethodPost)
		apiRouter.HandleFunc("/crypto/users/{userId}/assets", userAssetController.GetUserAssets).Methods(http.MethodGet)

	})
}
