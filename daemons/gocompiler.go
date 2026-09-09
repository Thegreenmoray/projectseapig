package daemons

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
)

// CommandRunner abstracts exec.Command for testing
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// RealCommandRunner is used in production
type RealCommandRunner struct{}

func (r *RealCommandRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// Thankfully this wasnt too complex, so using llms wasnt too detrimental to the learning process.
// the worst was the commands, which I would have likely had to look up anyway, but everything else was very striaghtforward.
// the daemons on the other hand are something i will likely need to struggle through on my own, as they are more complex and require a deeper understanding of the language and the problem domain. I will likely need to spend more time on those, and possibly seek help from others or use llms to assist me in understanding the concepts and implementation details.
// at least just one of them anyway.
type GoCompiler struct {
	ProjectPath  string // package path consisting of tests
	CompiledPath string //compiled binary path
	Timeout      time.Duration
	Runner       CommandRunner
}

func (gc *GoCompiler) getRunner() CommandRunner {
	if gc.Runner == nil {
		return &RealCommandRunner{}
	}
	return gc.Runner
}

func (gc *GoCompiler) Start() error {
	runner := gc.getRunner()
	_, err := runner.Run("go", "test", "-c", "-o", gc.CompiledPath, gc.ProjectPath)
	return err
}

func (gc *GoCompiler) Stop() error {
	return os.Remove(gc.CompiledPath)
}

func (gc *GoCompiler) RunTests(batchoftests []string) ([]runners.TestResult, error) {
	if len(batchoftests) == 0 {
		return nil, nil
	}

	// 1. Join test names with regex OR operator: ^(TestA|TestB|TestC)$
	regexPattern := fmt.Sprintf("^%s$", strings.Join(batchoftests, "|"))

	start := time.Now()
	output, _ := gc.getRunner().Run(gc.CompiledPath, "-test.v", "-test.run", regexPattern)
	duration := time.Since(start)

	// 2. Map results back to individual test names
	stdoutStr := string(output)
	var results []runners.TestResult

	for _, testName := range batchoftests {
		// Go test runner outputs "--- PASS: TestName" or "--- FAIL: TestName"
		passed := strings.Contains(stdoutStr, fmt.Sprintf("--- PASS: %s", testName))

		results = append(results, runners.TestResult{
			Testname:  testName,
			Passed:    passed,
			Stdout:    stdoutStr,                                   // or extract individual slice if parsing stdout
			Timetaken: duration / time.Duration(len(batchoftests)), // average time per test in batch
		})
	}

	return results, nil
}
