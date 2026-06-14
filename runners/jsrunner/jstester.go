package jsrunner

import "github.com/Justi/projectseapig/runners"

type JStester struct {
}

func (g JStester) Detect(projectPath string) bool {

	return true
}

func (g JStester) ListTests(projectPath string) ([]string, error) {

	return make([]string, 3), nil
}

func (g JStester) RunTest(testName string) (runners.TestResult, error) {

	return runners.TestResult{Testname: testName}, nil
}
