package pythonrunner

import "github.com/Justi/projectseapig/runners"

type Pythontester struct {
}

func (g Pythontester) Detect(projectPath string) bool {

	return true
}

func (g Pythontester) ListTests(projectPath string) ([]string, error) {

	return make([]string, 3), nil
}

func (g Pythontester) RunTest(testName string) (runners.TestResult, error) {

	return runners.TestResult{Testname: testName}, nil
}
