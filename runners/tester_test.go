package runners

import "testing"

// Create a unit test for Results() from testerinterface.go create a pig instance with the following traits a slice of Testresults with defualt values (make 10 of them), have 1 of those testresults fail, all others pass, use defualt for everything else, then call Results() and check that the Flakynessrate is 10% and that IsFlaky is true, also check that PassCount is 9 and FailCount is 1.
func TestResults(t *testing.T) {
	pig := &Pig{}
	pig.Run = make([]TestResult, 10)
	for i := 0; i < 10; i++ {
		pig.Run[i] = TestResult{Passed: true}
	}
	pig.Run[0].Passed = false

	Results(pig)

	if pig.Flakynessrate != 10.0 {
		t.Errorf("Expected flakiness rate to be 10.0, got %f", pig.Flakynessrate)
	}
	if !pig.IsFlaky {
		t.Errorf("Expected IsFlaky to be true, got %v", pig.IsFlaky)
	}
	if pig.PassCount != 9 {
		t.Errorf("Expected PassCount to be 9, got %d", pig.PassCount)
	}
	if pig.FailCount != 1 {
		t.Errorf("Expected FailCount to be 1, got %d", pig.FailCount)
	}
}
