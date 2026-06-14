package gorunner

import "github.com/Justi/projectseapig/runners"

type Gotester struct {
}

func (g Gotester) Detect(projectPath string) bool {

	return true
}

func (g Gotester) ListTests(projectPath string) ([]string, error) {

	return make([]string, 3), nil
}

func (g Gotester) RunTest(testName string) (runners.TestResult, error) {

	return runners.TestResult{Testname: testName}, nil
}
