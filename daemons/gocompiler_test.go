package daemons

import "testing"

type MockCommandRunner struct {
	// Map command/arguments or execution sequence to simulated output
	MockOutputs map[string]struct {
		Output []byte
		Err    error
	}
}

func (m *MockCommandRunner) Run(name string, args ...string) ([]byte, error) {
	// Key by the primary command or binary path being called
	if res, ok := m.MockOutputs[name]; ok {
		return res.Output, res.Err
	}
	return []byte("=== RUN   Test\n--- PASS: Test (0.00s)\nPASS"), nil
}

func TestGoCompiler_StartAndRunTests(t *testing.T) {
	mockRunner := &MockCommandRunner{
		MockOutputs: map[string]struct {
			Output []byte
			Err    error
		}{
			"go":               {Output: []byte("compilation successful"), Err: nil},
			"/tmp/mock_binary": {Output: []byte("=== RUN TestFoo\n--- PASS: TestFoo (0.01s)"), Err: nil},
		},
	}

	compiler := &GoCompiler{
		ProjectPath:  "./...",
		CompiledPath: "/tmp/mock_binary",
		Runner:       mockRunner,
	}

	// 1. Test compilation phase
	if err := compiler.Start(); err != nil {
		t.Fatalf("Start() failed unexpectedly: %v", err)
	}
	defer compiler.Stop()
	// 2. Test execution phase
	tests := []string{"TestFoo"}
	results, err := compiler.RunTests(tests)
	if err != nil {
		t.Fatalf("RunTests() failed unexpectedly: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Passed {
		t.Error("Expected test result to be marked as Passed")
	}

	if results[0].Testname != "TestFoo" {
		t.Errorf("Expected test name 'TestFoo', got %s", results[0].Testname)
	}
}
