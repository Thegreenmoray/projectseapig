package factory

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

func InitConfig() {
	viper.SetConfigName("seapig")
	viper.SetConfigType("yaml")

	// 1. Current working directory
	if cwd, err := os.Getwd(); err == nil {
		viper.AddConfigPath(cwd)
		viper.AddConfigPath(filepath.Dir(cwd)) // Parent dir
	}

	// 2. Binary execution path
	if exePath, err := os.Executable(); err == nil {
		viper.AddConfigPath(filepath.Dir(exePath))
	}

	// 3. Absolute path via source file location (Guarantees root during 'go test')
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		// factory package -> go up 1 level to project root
		projectRoot := filepath.Dir(filepath.Dir(filename))
		viper.AddConfigPath(projectRoot)
	}

	// 4. User home fallback
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".config", "seapig"))
	}

	// Default fallback values
	viper.SetDefault("workers", 10)
	viper.SetDefault("timeout", "5s")

	if err := viper.ReadInConfig(); err != nil {
		// Clear indication that YAML was NOT found
		fmt.Println("[CONFIG] ⚠️  No seapig.yaml found. Using hardcoded defaults.")
	} else {
		// Explicit validation message confirming the file loaded!
		fmt.Printf("[CONFIG] Loaded configuration from: %s\n", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}

	if viper.ConfigFileUsed() != "" {
		fmt.Printf("[CONFIG]  ├── Workers: %d\n", Cfg.Workers)
		fmt.Printf("[CONFIG]  └── Timeout: %s\n", Cfg.Timeout)
	}
}
