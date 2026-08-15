package gorunner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Gotester struct {
	BinPath     string // e.g., "go"
	ProjectPath string // e.g., "C:\Users\...\testfolder"
	BaseArgs    []string
	Timeout     time.Duration
	Env         []string
}

func (g *Gotester) ListTests(projectPath string) ([]string, error) {
	discoveryTimeout := g.Timeout
	if g.Timeout <= 0 {
		return nil, fmt.Errorf("Time is too short, please enter something larger than 0")
	}

	ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()

	bin := g.BinPath
	if bin == "" {
		bin = "go"
	}

	// Try scanning all subpackages
	cmd := exec.CommandContext(ctx, bin, "test", "-list", ".*", "./...")
	projectPath = filepath.Clean(projectPath)
	info, err := os.Stat(projectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project path does not exist: %s", projectPath)
		}
		return nil, fmt.Errorf("error accessing project path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project path is not a directory: %s", projectPath)
	}
	cmd.Dir = projectPath

	out, err := cmd.CombinedOutput()

	// Parse whatever output was generated
	lines := strings.Split(string(out), "\n")
	var tests []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if parts := strings.Fields(line); len(parts) > 0 && strings.HasPrefix(parts[0], "Test") {
			tests = append(tests, parts[0])
		}
	}

	// If subpackages failed, attempt listing the local directory package
	if len(tests) == 0 {
		cmdLocal := exec.CommandContext(ctx, bin, "test", "-list", ".*", ".")
		cmdLocal.Dir = projectPath
		outLocal, errLocal := cmdLocal.CombinedOutput()
		if errLocal != nil && len(outLocal) == 0 {
			return nil, fmt.Errorf("go test -list failed: %v | output: %s", err, string(out))
		}

		for _, line := range strings.Split(string(outLocal), "\n") {
			line = strings.TrimSpace(line)
			if parts := strings.Fields(line); len(parts) > 0 && strings.HasPrefix(parts[0], "Test") {
				tests = append(tests, parts[0])
			}
		}
	}

	if len(tests) == 0 {
		return nil, fmt.Errorf("no test functions starting with 'Test' found in %s", projectPath)
	}

	return tests, nil
}
