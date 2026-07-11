package jsrunner

import (
	"context"
	"os/exec"
	"strings"
	"time"

	//"strings"

	"github.com/Justi/projectseapig/runners"
)

type JStester struct {
	BinPath  string        // e.g., "/usr/local/go/bin/go" or just "go"
	BaseArgs []string      // e.g., []string{"test", "-run"}
	Timeout  time.Duration // Individual test execution timeout
	Env      []string      // Custom ENV vars for the test runner process
}

func (j *JStester) ListTests(projectPath string) ([]string, error) {
	// Use Jest's built-in flag via npx/npm to print individual test files
	ctx, cancel := context.WithTimeout(context.Background(), j.Timeout)
	defer cancel()

	// Using 'npx jest --listTests' is way more accurate than globbing directories
	cmd := exec.CommandContext(ctx, "npx", "jest", "--listTests")
	cmd.Dir = projectPath

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var tests []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Jest lists the full or relative file path to the test file
			tests = append(tests, line)
		}
	}

	return tests, nil
}

func (j *JStester) RunTest(testName string) (runners.TestResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), j.Timeout)
	defer cancel()

	// If using the factory defaults: bin="npm", args=["test", "--", testName]
	// This cleanly routes down to the package.json scripts configuration
	args := append(j.BaseArgs, testName)

	cmd := exec.CommandContext(ctx, j.BinPath, args...)
	if len(j.Env) > 0 {
		cmd.Env = j.Env
	}

	out, err := cmd.CombinedOutput()
	passed := err == nil

	if ctx.Err() == context.DeadlineExceeded {
		passed = false
		out = append(out, []byte("\n--- PROJECT SEAPIG: JavaScript execution timed out! ---")...)
	}

	return runners.TestResult{
		Testname: testName,
		Passed:   passed,
		Stdout:   string(out),
	}, nil
}
