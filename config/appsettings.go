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
	AppPort               string        `mapstructure:"appPort"  yaml:"appPort,omitempty"`
	ServiceName           string        `mapstructure:"serviceName"  yaml:"serviceName,omitempty"`
	DBConnectionString    string        `mapstructure:"DB_CONNECTION_URI"  yaml:"DB_CONNECTION_URI,omitempty"`
	BasePath              string        `mapstructure:"basePath"  yaml:"basePath,omitempty"`
	ServiceID             string        `mapstructure:"AUTHENTICATION_SERVICE_SERVICE_ID"  yaml:"serviceId,omitempty"`
	ServiceKey            string        `mapstructure:"AUTHENTICATION_SERVICE_TOKEN"  yaml:"serviceKey,omitempty"`
	AuthenticatorKey      string        `mapstructure:"SECURITY_BUNDLE_PUBLICKEY"  yaml:"authenticatorKey,omitempty"`
	AuthenticationService string        `mapstructure:"authenticationServiceURL"  yaml:"authenticationServiceURL,omitempty"`
	KeyManagementService  string        `mapstructure:"keyManagementServiceURL"  yaml:"keyManagementServiceURL,omitempty"`
	PurgeCacheInterval    time.Duration `mapstructure:"purgeCacheInterval"  yaml:"purgeCacheInterval,omitempty"`
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
	viper.BindEnv("DB_CONNECTION_URI")

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
