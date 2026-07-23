package gorunner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
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
	if discoveryTimeout < 60*time.Second {
		discoveryTimeout = 60 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()

	bin := g.BinPath
	if bin == "" {
		bin = "go"
	}

	// Try scanning all subpackages
	cmd := exec.CommandContext(ctx, bin, "test", "-list", ".*", "./...")
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

func (g *Gotester) RunTest(testName string) (runners.TestResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	bin := g.BinPath
	if bin == "" {
		bin = "go"
	}

	args := []string{"test", "-run", "^" + testName + "$", "-count=1", "./..."}

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = g.ProjectPath // Fixed: Now points to the actual project folder!

	if len(g.Env) > 0 {
		cmd.Env = g.Env
	}

	start := time.Now()
	out, err := cmd.CombinedOutput()
	passed := err == nil

	if ctx.Err() == context.DeadlineExceeded {
		passed = false
		out = append(out, []byte("\n--- PROJECT SEAPIG: Go test execution timed out! ---")...)
	}

	return runners.TestResult{
		Testname:  testName,
		Passed:    passed,
		Stdout:    string(out),
		Timetaken: time.Since(start),
	}, nil
}
