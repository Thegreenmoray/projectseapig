package gorunner

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type Gotester struct {
	BinPath  string        // e.g., "/usr/local/go/bin/go" or just "go"
	BaseArgs []string      // e.g., []string{"test", "-run"}
	Timeout  time.Duration // Individual test execution timeout
	Env      []string      // Custom ENV vars for the test runner process
}

func (g *Gotester) ListTests(projectPath string) ([]string, error) {
	// 1. Build the arguments dynamically: start with BaseArgs, append specific flags
	args := append(g.BaseArgs, "-list", ".", ".")

	// 2. Set up a context to respect the configured timeout
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	// 3. Use BinPath and your dynamic arguments
	cmd := exec.CommandContext(ctx, g.BinPath, args...)
	cmd.Dir = projectPath // Execute the command inside the target project directory
	if len(g.Env) > 0 {
		cmd.Env = g.Env
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var tests []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Test") {
			// Clean up any potential subtest variants or output clutter
			if parts := strings.Fields(line); len(parts) > 0 && strings.HasPrefix(parts[0], "Test") {
				tests = append(tests, parts[0])
			}
		}
	}

	return tests, nil
}

func (g *Gotester) RunTest(testName string) (runners.TestResult, error) {
	// 1. Build arguments dynamically: base args + targeting the specific test
	args := append(g.BaseArgs, "-run", "^"+testName+"$")

	// 2. Use CommandContext to enforce your 5s (or configured) timeout
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, g.BinPath, args...)
	if len(g.Env) > 0 {
		cmd.Env = g.Env
	}

	out, err := cmd.CombinedOutput()
	passed := err == nil

	// 3. Handle the scenario where the test was forcibly killed by the timeout context
	if ctx.Err() == context.DeadlineExceeded {
		passed = false
		out = append(out, []byte("\n--- PROJECT SEAPIG: Test execution timed out! ---")...)
	}

	return runners.TestResult{
		Testname: testName,
		Passed:   passed,
		Stdout:   string(out),
	}, nil
}
