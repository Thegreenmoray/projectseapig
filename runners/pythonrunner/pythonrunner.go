package pythonrunner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Pythontester struct {
	ProjectPath string // Target workspace path (e.g., "C:\Users\...\Testsinpython")
	BinPath     string // e.g., "pytest" or path to virtualenv pytest
	BaseArgs    []string
	Timeout     time.Duration
	Env         []string
}

func (g *Pythontester) ListTests(projectPath string) ([]string, error) {
	if g.Timeout <= 0 {
		return nil, fmt.Errorf("Time is too short, please enter something larger than 0")
	}

	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	bin := g.BinPath
	if bin == "" {
		bin = "pytest"
	}

	// --collect-only finds all tests. -q (quiet) strips unnecessary headers.
	args := append(g.BaseArgs, "--collect-only", "-q")

	cmd := exec.CommandContext(ctx, bin, args...)
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
	if len(g.Env) > 0 {
		cmd.Env = g.Env
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		// 1. Check specifically for context timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("test discovery timed out after %v. A Python test file may be executing heavy code or network calls at module import time instead of inside a fixture", g.Timeout)
		}

		// 2. Fall back to standard command failure (syntax error, missing pytest, etc.)
		return nil, fmt.Errorf("python test discovery failed: %v | output: %s", err, string(out))
	}

	lines := strings.Split(string(out), "\n")
	var tests []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "no tests ran") && strings.Contains(line, "::") {
			tests = append(tests, line)
		}
	}

	return tests, nil
}
