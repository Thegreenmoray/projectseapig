package factory

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Justi/projectseapig/runners/gorunner"
)

func TestInitConfigLoadsWorkingDirectoryConfig(t *testing.T) {
	previousConfig := Cfg
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		Cfg = previousConfig
		if err := os.Chdir(previousDirectory); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	}()

	configDirectory := t.TempDir()
	config := []byte("workers: 3\ntimeout: 45s\ndiscovery_timeout: 7m\ndebug: true\n")
	if err := os.WriteFile(filepath.Join(configDirectory, "seapig.yaml"), config, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(configDirectory); err != nil {
		t.Fatal(err)
	}

	if err := InitConfig(); err != nil {
		t.Fatalf("InitConfig() error = %v", err)
	}
	if Cfg.Workers != 3 || Cfg.Timeout != "45s" || Cfg.DiscoveryTimeout != "7m" || !Cfg.Debug {
		t.Fatalf("InitConfig() loaded unexpected values: %+v", Cfg)
	}
}

func TestGoRunnerUsesConfiguredDiscoveryTimeout(t *testing.T) {
	previousConfig := Cfg
	Cfg.Timeout = "1m"
	Cfg.DiscoveryTimeout = "7m"
	defer func() { Cfg = previousConfig }()

	runner, err := Testtype("go", ".")
	if err != nil {
		t.Fatalf("Testtype() error = %v", err)
	}
	goRunner, ok := runner.(*gorunner.Gotester)
	if !ok {
		t.Fatalf("Testtype() returned %T, want *gorunner.Gotester", runner)
	}
	if goRunner.Timeout != time.Minute || goRunner.DiscoveryTimeout != 7*time.Minute {
		t.Fatalf("Go runner timeouts = execution %v, discovery %v", goRunner.Timeout, goRunner.DiscoveryTimeout)
	}
}
