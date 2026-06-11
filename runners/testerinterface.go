package runners

import "time"

type TestRunner interface {
	Detect(projectPath string) bool
	ListTests(projectPath string) ([]string, error)
	RunTest(testName string) (error, TestResult)
}

type Flaky struct {
	Seriesoftests         []TestResult //would love to call this marine snow, but would be too confusing
	Oringialamountoftests int
	Flakynessrate         float64
	PassCount             int
	FailCount             int
}

type TestResult struct {
	Testname  string
	Passed    bool
	Timetaken time.Duration
	Exitcode  int
	Stdout    string
	Stderr    string
	//others later

}
