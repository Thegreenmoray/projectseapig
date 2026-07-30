package jsrunner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	//"strings"

	"github.com/Justi/projectseapig/runners"
)

type JStester struct {
	ProjectPath string // Target workspace path (e.g., "C:\Users\...\untitled3")
	BinPath     string // e.g., "npm" or "npx"
	BaseArgs    []string
	Timeout     time.Duration
	Env         []string
}

func (j *JStester) ListTests(projectPath string) ([]string, error) {
	if j.Timeout <= 0 {
		return nil, fmt.Errorf("Time is too short, please enter something larger than 0")
	}
	ctx, cancel := context.WithTimeout(context.Background(), j.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "npx", "jest", "--listTests")
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
	cmd.Dir = projectPath // Uses the passed parameter during discovery

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("JS test discovery failed: %v | output: %s", err, string(out))
	}

	lines := strings.Split(string(out), "\n")
	var tests []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			tests = append(tests, line)
		}
	}

	return tests, nil
}

func (j *JStester) RunTest(testName string) (runners.TestResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), j.Timeout)
	defer cancel()

	bin := j.BinPath
	if bin == "" {
		bin = "npm"
	}

	var args []string
	if strings.Contains(bin, "npm") {
		args = append([]string{"test", "--silent", "--"}, j.BaseArgs...)
		args = append(args, "-t", testName, "--runInBand", "--no-coverage")
	} else {
		args = append(j.BaseArgs, "-t", testName, "--runInBand", "--no-coverage", "--silent")
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = j.ProjectPath // CRITICAL: Sets working dir to the project root

	env := os.Environ()
	env = append(env, "NODE_ENV=test")
	if len(j.Env) > 0 {
		env = append(env, j.Env...)
	}
	cmd.Env = env

	start := time.Now()
	out, err := cmd.CombinedOutput()
	passed := err == nil

	if ctx.Err() == context.DeadlineExceeded {
		passed = false
		out = append(out, []byte("\n--- PROJECT SEAPIG: JavaScript execution timed out! ---")...)
	}

	return runners.TestResult{
		Testname:  testName,
		Passed:    passed,
		Stdout:    string(out),
		Timetaken: time.Since(start),
	}, nil
}
