package config

import (
	"fmt"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

//Data : config data
type Data struct {
	LogFile            string `mapstructure:"logfile"  yaml:"logfile,omitempty"`
	AppPort            string `mapstructure:"appPort"  yaml:"appPort,omitempty"`
	LogFolder          string `mapstructure:"logFolder"  yaml:"logFolder,omitempty"`
	AppName            string `mapstructure:"appName"  yaml:"appName,omitempty"`
	DBConnectionString string `mapstructure:"dbConnectionString"  yaml:"dbConnectionString,omitempty"`
}

//Init : initialize data
func (c *Data) Init(configDir string) {

	dir, dirErr := os.Getwd()
	if dirErr != nil {
		log.Printf("Cannot set default input/output directory to the current working directory >> %s", dirErr)
	}

	viper.SetEnvPrefix("was") // wPrefix all env variable with WAS(Wallet adapter Service)
	viper.AutomaticEnv()
	viper.BindEnv("appPort")

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
