package pythonrunner

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type Pythontester struct {
	BinPath  string        // e.g., "/usr/local/go/bin/go" or just "go"
	BaseArgs []string      // e.g., []string{"test", "-run"}
	Timeout  time.Duration // Individual test execution timeout
	Env      []string      // Custom ENV vars for the test runner process
}

func (g *Pythontester) ListTests(projectPath string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	// --collect-only finds all tests. -q (quiet) strips unnecessary headers.
	args := append(g.BaseArgs, "--collect-only", "-q")

	cmd := exec.CommandContext(ctx, g.BinPath, args...)
	cmd.Dir = projectPath
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
		// pytest output lines look like: tests/test_math.py::test_addition
		// We filter out tracking metrics or empty lines at the bottom of quiet mode
		if line != "" && !strings.Contains(line, "no tests ran") && strings.Contains(line, "::") {
			tests = append(tests, line)
		}
	}

	return tests, nil
}

func (g *Pythontester) RunTest(testName string) (runners.TestResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	// testName will look exactly like: tests/test_math.py::test_addition
	args := append(g.BaseArgs, testName)

	cmd := exec.CommandContext(ctx, g.BinPath, args...)
	if len(g.Env) > 0 {
		cmd.Env = g.Env
	}

	out, err := cmd.CombinedOutput()
	passed := err == nil

	if ctx.Err() == context.DeadlineExceeded {
		passed = false
		out = append(out, []byte("\n--- PROJECT SEAPIG: Python execution timed out! ---")...)
	}

	return runners.TestResult{
		Testname: testName,
		Passed:   passed,
		Stdout:   string(out),
	}, nil
}
