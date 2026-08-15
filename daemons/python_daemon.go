package daemons

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type PythonDaemon struct {
	daemonBase  DaemonBase
	Socketpath  string        // Path to the Unix socket for communication with the Python daemon
	Timeout     time.Duration //until a batch of tests are timed out
	DeamonPath  string        //where the deamon is located
	ProjectRoot string
}

func (p *PythonDaemon) RunTests(testName string) ([]runners.TestResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
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
