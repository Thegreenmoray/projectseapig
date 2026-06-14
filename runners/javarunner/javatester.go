package javarunner

import "github.com/Justi/projectseapig/runners"

type Javatester struct {
}

func (g Javatester) Detect(projectPath string) bool {

	return true
}

func (g Javatester) ListTests(projectPath string) ([]string, error) {

	return make([]string, 3), nil
}

func (g Javatester) RunTest(testName string) (runners.TestResult, error) {

	return runners.TestResult{Testname: testName}, nil
}
