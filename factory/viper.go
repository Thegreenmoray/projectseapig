package factory

import (
	"log"

	"github.com/spf13/viper"
)

func InitConfig() {
	viper.SetConfigName("seapig")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/seapig")

	// Defaults
	viper.SetDefault("workers", 10)
	viper.SetDefault("timeout", "5s")
	viper.SetDefault("debug", false)
	viper.SetDefault("language", "")

	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("no config file found, using defaults")
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}
}
