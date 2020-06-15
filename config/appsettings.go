package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

//Data : config data
type Data struct {
	AppPort                     string        `mapstructure:"appPort"  yaml:"appPort,omitempty"`
	ServiceName                 string        `mapstructure:"serviceName"  yaml:"serviceName,omitempty"`
	DBHost                      string        `mapstructure:"DB_HOST"  yaml:"DB_HOST,omitempty"`
	DBUser                      string        `mapstructure:"DB_USER"  yaml:"DB_USER,omitempty"`
	DBPassword                  string        `mapstructure:"DB_PASSWORD"  yaml:"DB_PASSWORD,omitempty"`
	DBName                      string        `mapstructure:"DB_NAME"  yaml:"DB_NAME,omitempty"`
	BasePath                    string        `mapstructure:"basePath"  yaml:"basePath,omitempty"`
	ServiceID                   string        `mapstructure:"AUTHENTICATION_SERVICE_SERVICE_ID"  yaml:"AUTHENTICATION_SERVICE_SERVICE_ID,omitempty"`
	ServiceKey                  string        `mapstructure:"AUTHENTICATION_SERVICE_TOKEN"  yaml:"AUTHENTICATION_SERVICE_TOKEN,omitempty"`
	AuthenticatorKey            string        `mapstructure:"SECURITY_BUNDLE_PUBLICKEY"  yaml:"SECURITY_BUNDLE_PUBLICKEY,omitempty"`
	AuthenticationService       string        `mapstructure:"authenticationServiceURL"  yaml:"authenticationServiceURL,omitempty"`
	KeyManagementService        string        `mapstructure:"keyManagementServiceURL"  yaml:"keyManagementServiceURL,omitempty"`
	CryptoAdapterService        string        `mapstructure:"cryptoAdapterServiceURL"  yaml:"cryptoAdapterServiceURL,omitempty"`
	LockerService               string        `mapstructure:"lockerServiceURL"  yaml:"lockerServiceURL,omitempty"`
	LockerPrefix                string        `mapstructure:"lockerServicePrefix"  yaml:"lockerServicePrefix,omitempty"`
	DepositWebhookURL           string        `mapstructure:"depositWebhookURL"  yaml:"depositWebhookURL,omitempty"`
	WithdrawToHotWalletUrl      string        `mapstructure:"withdrawToHotWalletUrl"  yaml:"withdrawToHotWalletUrl,omitempty"`
	NotificationServiceUrl      string        `mapstructure:"notificationServiceUrl"  yaml:"notificationServiceUrl,omitempty"`
	ColdWalletEmail             string        `mapstructure:"coldWalletEmail"  yaml:"coldWalletEmail,omitempty"`
	ColdWalletEmailTemplateId   string        `mapstructure:"coldWalletEmailTemplateId"  yaml:"coldWalletEmailTemplateId,omitempty"`
	BtcSlipValue                string        `mapstructure:"BTC_SLIP_VALUE"  yaml:"BTC_SLIP_VALUE,omitempty"`
	BnbSlipValue                string        `mapstructure:"BNB_SLIP_VALUE"  yaml:"BNB_SLIP_VALUE,omitempty"`
	EthSlipValue                string        `mapstructure:"ETH_SLIP_VALUE"  yaml:"ETH_SLIP_VALUE,omitempty"`
	PurgeCacheInterval          time.Duration `mapstructure:"purgeCacheInterval"  yaml:"purgeCacheInterval,omitempty"`
	RequestTimeout              time.Duration `mapstructure:"requestTimeout"  yaml:"requestTimeout,omitempty"`
	ExpireCacheDuration         time.Duration `mapstructure:"expireCacheDuration"  yaml:"expireCacheDuration,omitempty"`
	SweepFeePercentageThreshold int64         `mapstructure:"sweepFeePercentageThreshold"  yaml:"sweepFeePercentageThreshold,omitempty"`
	MaxIdleConns                int           `mapstructure:"maxIdleConns"  yaml:"maxIdleConns,omitempty"`
	MaxOpenConns                int           `mapstructure:"maxOpenConns"  yaml:"maxOpenConns,omitempty"`
	ConnMaxLifetime             int           `mapstructure:"connMaxLifetime"  yaml:"connMaxLifetime,omitempty"`
	FloatPercentage             int           `mapstructure:"floatPercentage"  yaml:"floatPercentage,omitempty"`
	EnableFloatManager          bool          `mapstructure:"enableFloatManager"  yaml:"enableFloatManager,omitempty"`
	SweepCronInterval           string        `mapstructure:"sweepCronInterval"  yaml:"sweepCronInterval,omitempty"`
	FloatCronInterval           string        `mapstructure:"floatCronInterval"  yaml:"floatCronInterval,omitempty"`
	DBMigrationPath             string        `mapstructure:"dbMigrationPath"  yaml:"dbMigrationPath,omitempty"`
	SentryDsn                   string        `mapstructure:"SENTRY_DSN"  yaml:"SENTRY_DSN,omitempty"`
	SENTRY_ENVIRONMENT          string        `mapstructure:"SENTRY_ENVIRONMENT"  yaml:"SENTRY_ENVIRONMENT,omitempty"`
}

//Init : initialize data
func (c *Data) Init(configDir string) {

	dir, dirErr := os.Getwd()
	if dirErr != nil {
		log.Printf("Cannot set default input/output directory to the current working directory >> %s", dirErr)
	}

	// viper.SetEnvPrefix("") // wPrefix all env variable with WAS(Wallet adapter Service) i.e WAS-APPPORT
	viper.AutomaticEnv()
	viper.BindEnv("AUTHENTICATION_SERVICE_SERVICE_ID")
	viper.BindEnv("AUTHENTICATION_SERVICE_TOKEN")
	viper.BindEnv("SECURITY_BUNDLE_PUBLICKEY")
	viper.BindEnv("DB_HOST")
	viper.BindEnv("DB_USER")
	viper.BindEnv("DB_PASSWORD")
	viper.BindEnv("DB_NAME")
	viper.BindEnv("SENTRY_ENVIRONMENT")

	viper.SetConfigName("config")
	viper.AddConfigPath("../")
	viper.AddConfigPath(dir)
	viper.AddConfigPath(configDir)
	viper.WatchConfig()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(fmt.Errorf("\n Configuration file not found >>%s ", err))
		} else {
			panic(fmt.Errorf("\n fatal error: could not read from config file >>%s ", err))
		}
	}

	viper.OnConfigChange(func(e fsnotify.Event) {
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				panic(fmt.Errorf("\n Configuration file not found >>%s ", err))
			} else {
				panic(fmt.Errorf("\n fatal error: could not read from config file >>%s ", err))
			}
		}
		viper.Unmarshal(c)
		fmt.Println("Config file changed:", e.Name)
	})

	viper.Unmarshal(c)
	log.Println("App configuration loaded successfully!")
}

var SupportedAssets = [...]string{
	"BTC",
	"BNB",
	"ETH",
}
