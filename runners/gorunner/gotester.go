package gorunner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Justi/projectseapig/runners"
)

type Gotester struct {
}

func (g Gotester) Detect(projectPath string) bool {
	//stats as in status of the file
	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); err == nil {
		return true
	}
	//Glob is equvient to ls
	matches, _ := filepath.Glob(filepath.Join(projectPath, "*_test.go"))
	return len(matches) > 0
}

func (g Gotester) ListTests(projectPath string) ([]string, error) {
	//basic command line
	cmd := exec.Command("go", "test", "-list", ".", projectPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var tests []string
	//equventanet to an enhanced for loop
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Test") {
			tests = append(tests, line)
		}
	}

	return tests, nil
}

func (g Gotester) RunTest(testName string) (runners.TestResult, error) {

	cmd := exec.Command("go", "test", "-run", "^"+testName+"$")
	out, err := cmd.CombinedOutput()

	passed := err == nil

	return runners.TestResult{
		Testname: testName,
		Passed:   passed,
		Stdout:   string(out),
	}, nil
}
