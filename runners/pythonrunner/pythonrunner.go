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

func (g *Pythontester) Detect(projectPath string) (int, error) {
	score := 0

	// Strong indicators
	if _, err := os.Stat(filepath.Join(projectPath, "pyproject.toml")); err == nil {
		score += 10
	}
	if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		score += 8
	}
	if _, err := os.Stat(filepath.Join(projectPath, "Pipfile")); err == nil {
		score += 8
	}
	if _, err := os.Stat(filepath.Join(projectPath, "setup.py")); err == nil {
		score += 8
	}

	// Test file patterns
	patterns := []string{"test_*.py", "*_test.py"}
	for _, ptn := range patterns {
		matches, _ := filepath.Glob(filepath.Join(projectPath, ptn))
		if len(matches) > 0 {
			score += 5
		}
	}

	// Any .py files at all (fallback)
	pyFiles, _ := filepath.Glob(filepath.Join(projectPath, "*.py"))
	if len(pyFiles) > 0 {
		score += 3
	}

	return score, nil
}

func (g *Pythontester) ListTests(projectPath string) ([]string, error) {

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
