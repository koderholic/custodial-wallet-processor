package test

import (
	"net/http"
	"sync"
	Config "wallet-adapter/config"
	"wallet-adapter/middlewares"
	"wallet-adapter/utility"

	"wallet-adapter/controllers"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	validation "gopkg.in/go-playground/validator.v9"
)

var (
	once sync.Once
)

func startUp() (Config.Data, *utility.Logger, http.Handler) {

	config := Config.Data{
		AppPort:            "9000",
		ServiceName:        "wallet-adapter",
		BasePath:           "/api/v1",
		AuthenticatorKey:   "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
		PurgeCacheInterval: 5,
	}

	logger := utility.NewLogger()
	router := mux.NewRouter()
	validator := validation.New()

	RegisterRoutes(router, validator, config, logger)

	middleware := middlewares.NewMiddleware(logger, config, router).ValidateAuthToken().LogAPIRequests().Build()

	return config, logger, middleware
}

var config, logger, router = startUp()

func RegisterRoutes(router *mux.Router, validator *validation.Validate, config Config.Data, logger *utility.Logger) {

	once.Do(func() {
		baseRepository := BaseRepository{Logger: logger, Config: config}
		userAssetRepository := UserAssetRepository{BaseRepository: baseRepository}

		controller := controllers.NewController(logger, config, validator, &baseRepository)
		userAssetController := controllers.NewUserAssetController(logger, config, validator, &userAssetRepository)

		basePath := config.BasePath

		apiRouter := router.PathPrefix(basePath).Subrouter()
		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// General Routes
		apiRouter.HandleFunc("/crypto/ping", controller.Ping).Methods(http.MethodGet)

		// Asset Routes
		apiRouter.HandleFunc("/crypto/assets", controller.FetchAllAssets).Methods(http.MethodGet)
		apiRouter.HandleFunc("/crypto/assets/supported", controller.FetchSupportedAssets).Methods(http.MethodGet)
		apiRouter.HandleFunc("/crypto/assets/{assetId}", controller.GetAsset).Methods(http.MethodGet)

		// User Asset Routes
		apiRouter.HandleFunc("/crypto/users/assets", userAssetController.CreateUserAssets).Methods(http.MethodPost)
		apiRouter.HandleFunc("/crypto/users/{userId}/assets", userAssetController.GetUserAssets).Methods(http.MethodGet)

	})

	logger.Info("App routes registered successfully!")
}
