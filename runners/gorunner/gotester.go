package gorunner

import (
	"os/exec"
	"strings"

	"github.com/Justi/projectseapig/runners"
)

type Gotester struct {
}

func (g *Gotester) Detect(projectPath string) (int, error) {
	score := 0

	return score, nil
}

func (g *Gotester) ListTests(projectPath string) ([]string, error) {
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

func (g *Gotester) RunTest(testName string) (runners.TestResult, error) {

	cmd := exec.Command("go", "test", "-run", "^"+testName+"$")
	out, err := cmd.CombinedOutput()

	passed := err == nil

	return runners.TestResult{
		Testname: testName,
		Passed:   passed,
		Stdout:   string(out),
	}, nil
}
