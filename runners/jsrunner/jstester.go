package jsrunner

import (
	//"os"
	//"os/exec"
	"path/filepath"
	//"strings"

	"github.com/Justi/projectseapig/runners"
)

type JStester struct {
}

func (g *JStester) ListTests(projectPath string) ([]string, error) {

	patterns := []string{
		"*.test.js", "*.spec.js",
		"*.test.mjs", "*.spec.mjs",
	}

	var tests []string
	for _, p := range patterns {
		matches, _ := filepath.Glob(filepath.Join(projectPath, p))
		tests = append(tests, matches...)
	}

	return tests, nil

	/*above is stub this will be the real logic
	cmd := exec.Command("npx", "jest", "--listTests")
	cmd.Dir = projectPath

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var tests []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, ".test.js") || strings.HasSuffix(line, ".spec.js") ||
			strings.HasSuffix(line, ".test.mjs") || strings.HasSuffix(line, ".spec.mjs") {
			tests = append(tests, line)
		}
	}

	return tests, nil*/
}

func (j *JStester) RunTest(testName string) (runners.TestResult, error) {

	return runners.TestResult{
		Testname: testName,
		Passed:   true,
		Stdout:   "simulated JS test run",
	}, nil

	/*above is stub this will be the real logic
	cmd := exec.Command("npx", "jest", testName)
	out, err := cmd.CombinedOutput()
	passed := err == nil

	return runners.TestResult{
		Testname: testName,
		Passed:   passed,
		Stdout:   string(out),
	}, nil*/
}
