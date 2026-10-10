package factory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

func InitConfig() error {
	configLoader := viper.New()
	configLoader.SetConfigName("seapig")
	configLoader.SetConfigType("yaml")

	// 1. Current working directory
	if cwd, err := os.Getwd(); err == nil {
		configLoader.AddConfigPath(cwd)
		configLoader.AddConfigPath(filepath.Dir(cwd)) // Parent dir
	}

	// 2. Binary execution path
	if exePath, err := os.Executable(); err == nil {
		configLoader.AddConfigPath(filepath.Dir(exePath))
	}

	// 3. Absolute path via source file location (Guarantees root during 'go test')
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		// factory package -> go up 1 level to project root
		projectRoot := filepath.Dir(filepath.Dir(filename))
		configLoader.AddConfigPath(projectRoot)
	}

	// 4. User home fallback
	if home, err := os.UserHomeDir(); err == nil {
		configLoader.AddConfigPath(filepath.Join(home, ".config", "seapig"))
	}

	// Default fallback values
	configLoader.SetDefault("workers", 25)
	configLoader.SetDefault("timeout", "5s")
	configLoader.SetDefault("discovery_timeout", "10m")
	configLoader.SetDefault("debug", false)

	err := configLoader.ReadInConfig()
	if err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("failed to read seapig configuration: %w", err)
		}

		configLoader.SetConfigName(".seapig")
		err = configLoader.ReadInConfig()
		if err != nil {
			if !errors.As(err, &notFound) {
				return fmt.Errorf("failed to read seapig configuration: %w", err)
			}
			fmt.Println("[CONFIG] No seapig.yaml found. Using hardcoded defaults.")
		}
	}

	if err := configLoader.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if configPath := configLoader.ConfigFileUsed(); configPath != "" {
		fmt.Printf("[CONFIG] Loaded configuration from: %s\n", configPath)
		fmt.Printf("[CONFIG]  ├── Workers: %d\n", Cfg.Workers)
		fmt.Printf("[CONFIG]  ├── Timeout: %s\n", Cfg.Timeout)
		fmt.Printf("[CONFIG]  └── Discovery timeout: %s\n", Cfg.DiscoveryTimeout)
	}
	return nil
}
