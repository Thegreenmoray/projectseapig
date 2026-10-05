package daemons

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"
)

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

type SequencedCommandRunner struct {
	Responses []struct {
		Output []byte
		Err    error
	}
	Calls [][]string
}

func (r *SequencedCommandRunner) Run(name string, args ...string) ([]byte, error) {
	r.Calls = append(r.Calls, append([]string{name}, args...))
	response := r.Responses[0]
	r.Responses = r.Responses[1:]
	return response.Output, response.Err
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

func TestGoCompilerRunsRepeatedTestsIndependently(t *testing.T) {
	runner := &SequencedCommandRunner{Responses: []struct {
		Output []byte
		Err    error
	}{
		{Output: []byte("--- PASS: TestRepeat (0.00s)\nPASS\n")},
		{Output: []byte("--- FAIL: TestRepeat (0.00s)\nFAIL\n"), Err: errors.New("exit status 1")},
		{Output: []byte("--- PASS: TestOther (0.00s)\nPASS\n")},
	}}
	compiler := &GoCompiler{CompiledPath: "compiled-tests", Runner: runner}

	results, err := compiler.RunTests([]string{"TestRepeat", "TestRepeat", "TestOther"})
	if err != nil {
		t.Fatalf("RunTests() failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 iteration results, got %d", len(results))
	}
	wantPassed := []bool{true, false, true}
	for index, passed := range wantPassed {
		if results[index].Passed != passed {
			t.Errorf("result %d passed = %t, want %t", index, results[index].Passed, passed)
		}
	}
	if len(runner.Calls) != 3 {
		t.Fatalf("expected one compiled-binary invocation per selector, got %d", len(runner.Calls))
	}
	if results[0].Testname != results[1].Testname {
		t.Errorf("repeated iterations should retain the same test name: %q != %q", results[0].Testname, results[1].Testname)
	}
}

func TestGoCompilerUsesCompiledBinaryForDiscoveredPackage(t *testing.T) {
	runner := &SequencedCommandRunner{Responses: []struct {
		Output []byte
		Err    error
	}{
		{Output: []byte("root package compiled")},
		{Output: []byte("sub-package compiled")},
		{Output: []byte("--- PASS: TestSub (0.00s)\nPASS\n")},
	}}
	compiler := &GoCompiler{
		ProjectPath:  ".",
		CompiledPath: filepath.Join(t.TempDir(), "seapig_test_runner"),
		Runner:       runner,
	}
	if err := compiler.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer compiler.Stop()

	selector := "example.com/project/sub::TestSub"
	results, err := compiler.RunTests([]string{selector})
	if err != nil {
		t.Fatalf("RunTests() failed: %v", err)
	}
	if len(results) != 1 || !results[0].Passed || results[0].Testname != selector {
		t.Fatalf("unexpected package test result: %+v", results)
	}
	if len(runner.Calls) != 3 {
		t.Fatalf("expected root compile, package compile, and test run; got %d calls", len(runner.Calls))
	}
	packageCompile := runner.Calls[1]
	if len(packageCompile) < 6 || packageCompile[0] != "go" || packageCompile[1] != "test" || packageCompile[5] != "example.com/project/sub" {
		t.Fatalf("unexpected package compile command: %#v", packageCompile)
	}
	if runner.Calls[2][0] != packageCompile[4] {
		t.Fatalf("test ran with %q, want package binary %q", runner.Calls[2][0], packageCompile[4])
	}
}

func TestGoCompilerExecutesPackageQualifiedTest(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository root")
	}
	repoRoot := filepath.Dir(filepath.Dir(testFile))
	compiledPath := filepath.Join(t.TempDir(), "seapig_test_runner")
	if runtime.GOOS == "windows" {
		compiledPath += ".exe"
	}
	compiler := &GoCompiler{ProjectPath: repoRoot, CompiledPath: compiledPath}
	if err := compiler.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer compiler.Stop()

	results, err := compiler.RunTests([]string{"github.com/Justi/projectseapig/factory::TestColors"})
	if err != nil {
		t.Fatalf("RunTests() failed: %v", err)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Fatalf("package-qualified test did not pass: %+v", results)
	}
}

func TestGoTestResultMatchesExactTestName(t *testing.T) {
	output := []byte("--- PASS: TestExampleExtra (0.00s)\nPASS\n")
	if passed, reported := goTestResult(output, "TestExample"); reported || passed {
		t.Fatalf("prefix match should not produce a result, got passed=%t reported=%t", passed, reported)
	}
}
