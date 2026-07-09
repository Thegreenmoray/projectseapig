package pythonrunner

import (
	//"os"
	//"os/exec"
	"path/filepath"
	//"strings"

	"github.com/Justi/projectseapig/runners"
)

type Pythontester struct {
}

func (g *Pythontester) ListTests(projectPath string) ([]string, error) {

	patterns := []string{"test_*.py", "*_test.py"}
	var tests []string

	for _, p := range patterns {
		matches, _ := filepath.Glob(filepath.Join(projectPath, p))
		for _, m := range matches {
			tests = append(tests, filepath.Base(m))
		}
	}

	return tests, nil
}

func (g *Pythontester) RunTest(testName string) (runners.TestResult, error) {

	return runners.TestResult{
		Testname: testName,
		Passed:   true,
		Stdout:   "simulated Python test run",
	}, nil

	/* above is stub this will be the real logic
	cmd := exec.Command("pytest", testName)
	out, err := cmd.CombinedOutput()
	passed := err == nil

	return runners.TestResult{
		Testname: testName,
		Passed:   passed,
		Stdout:   string(out),
	}, nil*/
}
