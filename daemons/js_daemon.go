package daemons

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type JsDaemon struct {
	daemonBase  DaemonBase
	Socketpath  string        // Path to the Unix socket for communication with the JS daemon
	Timeout     time.Duration //until a batch of tests are timed out
	DeamonPath  string        //where the deamon is located
	ProjectRoot string        //tests are located
	NodePath    string
	IsTS        bool
}

func (j *JsDaemon) RunTest(testName string) ([]runners.TestResult, error) {
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
