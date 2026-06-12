package runners

import "time"

type TestRunner interface {
	Detect(projectPath string) bool
	ListTests(projectPath string) ([]string, error)
	RunTest(testName string) (error, TestResult)
}

type Pig struct {
	Run           []TestResult //would love to call this marine snow, but would be too confusing
	Flakynessrate float64
	PassCount     int
	FailCount     int
}

type TestResult struct {
	Testname  string
	Passed    bool
	Timetaken time.Duration
	Timestamp time.Time
	Exitcode  int
	Stdout    string
	Stderr    string
	Metadata  map[string]string //this stores any errors,callbacks, or panics from the test langugaes
	//others later

}
