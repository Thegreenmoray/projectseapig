package factory

import (
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func InitConfig() {
	viper.SetConfigName("seapig")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// Dynamically fetch the real home directory
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".config", "seapig"))
	}

	// Defaults
	viper.SetDefault("workers", 10)
	viper.SetDefault("timeout", "5s")
	viper.SetDefault("debug", false)
	viper.SetDefault("language", "")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("no config file found, using defaults")
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}
}
