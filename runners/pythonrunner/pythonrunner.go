package pythonrunner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type Pythontester struct {
	ProjectPath string // Target workspace path (e.g., "C:\Users\...\Testsinpython")
	BinPath     string // e.g., "pytest" or path to virtualenv pytest
	BaseArgs    []string
	Timeout     time.Duration
	Env         []string
}

func (g *Pythontester) ListTests(projectPath string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	bin := g.BinPath
	if bin == "" {
		bin = "pytest"
	}

	// --collect-only finds all tests. -q (quiet) strips unnecessary headers.
	args := append(g.BaseArgs, "--collect-only", "-q")

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = projectPath
	if len(g.Env) > 0 {
		cmd.Env = g.Env
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
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

func (g *Pythontester) RunTest(testName string) (runners.TestResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	bin := g.BinPath
	if bin == "" {
		bin = "pytest"
	}

	// High-performance Pytest CLI flags:
	defaultArgs := []string{"-q", "--no-header", "--no-summary"}
	args := append(defaultArgs, g.BaseArgs...)
	args = append(args, testName)

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = g.ProjectPath // CRITICAL FIX: Directs execution to target project folder

	// Environment Setup: Inject PYTHONDONTWRITEBYTECODE=1 to eliminate pycache disk writes
	env := os.Environ()
	env = append(env, "PYTHONDONTWRITEBYTECODE=1")
	if len(g.Env) > 0 {
		env = append(env, g.Env...)
	}
	cmd.Env = env

	start := time.Now()
	out, err := cmd.CombinedOutput()
	passed := err == nil

	if ctx.Err() == context.DeadlineExceeded {
		passed = false
		out = append(out, []byte("\n--- PROJECT SEAPIG: Python execution timed out! ---")...)
	}

	return runners.TestResult{
		Testname:  testName,
		Passed:    passed,
		Stdout:    string(out),
		Timetaken: time.Since(start),
	}, nil
}
