package pythonrunner

import (
	"os"
	//"os/exec"
	"path/filepath"
	//"strings"

	"github.com/Justi/projectseapig/runners"
)

type Pythontester struct {
}

func (g Pythontester) Detect(projectPath string) bool {

	if _, err := os.Stat(filepath.Join(projectPath, "pyproject.toml")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		return true
	}

	patterns := []string{"test_*.py", "*_test.py"}

	for _, ptn := range patterns {
		matches, _ := filepath.Glob(filepath.Join(projectPath, ptn))
		if len(matches) > 0 {
			return true
		}
	}

	return false
}

func (g Pythontester) ListTests(projectPath string) ([]string, error) {

	patterns := []string{"test_*.py", "*_test.py"}

	var tests []string
	for _, ptn := range patterns {
		matches, _ := filepath.Glob(filepath.Join(projectPath, ptn))
		tests = append(tests, matches...)
	}

	return tests, nil

	/* above is stub this will be the real logic
	cmd := exec.Command("pytest", "--collect-only", "-q")
	cmd.Dir = projectPath

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var tests []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "::") { // pytest format: file.py::TestClass::test_method
			tests = append(tests, line)
		}
	}

	return tests, nil
	*/
}

func (g Pythontester) RunTest(testName string) (runners.TestResult, error) {

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
