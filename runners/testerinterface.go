package runners

import "time"

type TestRunner interface {
	Detect(projectPath string) (int, error)
	ListTests(projectPath string) ([]string, error)
	RunTest(testName string) (TestResult, error)
}

type Pig struct {
	Run           []TestResult //would love to call this marine snow, but would be too confusing
	Flakynessrate float64
	PassCount     int
	FailCount     int
	IsFlaky       bool
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

func results(seapig *Pig) {
	ran := len(seapig.Run)
	if ran < 0 {
		return
	}
	pass := 0
	fail := 0
	for _, test := range seapig.Run {
		if test.Passed {
			pass++
		} else {
			fail++
		}
	}
	// Flakiness calculation: failed runs / total runs
	seapig.Flakynessrate = (float64(seapig.FailCount) / float64(ran)) * 100

	// A test is technically flaky if it has both passed AND failed in the same batch
	seapig.IsFlaky = seapig.PassCount > 0 && seapig.FailCount > 0
}
