package runners

import "time"

type TestRunner interface {
	ListTests(projectPath string) ([]string, error)
	RunTest(testName string) (TestResult, error)
}

type Pig struct {
	Run           []TestResult //would love to call this marine snow, but would be too confusing
	Flakynessrate float64
	PassCount     int
	FailCount     int
	IsFlaky       bool
	Testname      string
}

type TestResult struct {
	Testname  string            `json:"test_name"`
	Passed    bool              `json:"passed"`
	Timetaken time.Duration     `json:"time_taken"`
	Timestamp time.Time         `json:"timestamp"`
	Exitcode  int               `json:"exit_code"`
	Stdout    string            `json:"stdout"`
	Stderr    string            `json:"stderr"`
	Metadata  map[string]string `json:"metadata"`
}

func Results(seapig *Pig) {
	ran := len(seapig.Run)
	if ran < 0 {
		return
	}

	for _, test := range seapig.Run {
		if test.Passed {
			seapig.PassCount++
		} else {
			seapig.FailCount++
		}
	}
	// Flakiness calculation: failed runs / total runs
	seapig.Flakynessrate = (float64(seapig.FailCount) / float64(ran)) * 100

	// A test is technically flaky if it has both passed AND failed in the same batch
	seapig.IsFlaky = seapig.PassCount > 0 && seapig.FailCount > 0
}
