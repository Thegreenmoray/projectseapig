package cmd

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Justi/projectseapig/runners"
	"github.com/spf13/cobra"
)

// 1. Create a lightweight mock that satisfies the runners.TestRunner interface
type MockPigRunner struct {
	ShouldFailExecution bool
}

func (m *MockPigRunner) ListTests(projectPath string) ([]string, error) {
	// Return a single predictable test target name
	return []string{"mock_test_case"}, nil
}

func (m *MockPigRunner) RunTest(testName string) (runners.TestResult, error) {
	if m.ShouldFailExecution {
		return runners.TestResult{
			Testname:  testName,
			Passed:    false,
			Stdout:    "Mock failure output",
			Timetaken: 10 * time.Millisecond,
		}, nil
	}
	return runners.TestResult{
		Testname:  testName,
		Passed:    true,
		Stdout:    "Mock success output",
		Timetaken: 10 * time.Millisecond,
	}, nil
}

// 2. The Unit Test Suite
func TestRunCmd_Success(t *testing.T) {
	// Intercept Cobra's output buffer
	buf := new(bytes.Buffer)
	runCmd.SetOut(buf)
	runCmd.SetErr(buf)

	// Explicitly set the flag value programmatically for the test execution
	lang = "go"

	// Inject our Mock runner instead of letting the factory call the OS
	// To make this work seamlessly, we can temporarily wrap the factory target
	// or invoke the functions directly.

	// Let's execute the command using Cobra's executor engine
	runCmd.SetArgs([]string{"--lang", "go"})

	// Since os.Exit(0) or os.Exit(1) will kill our test run completely,
	// we want to test the internal worker routines directly to verify
	// our channel boundaries and WaitGroups remain stable!
	t.Run("Pipeline Concurrency Sanity Check", func(t *testing.T) {
		mockPig := &MockPigRunner{ShouldFailExecution: false}
		jobs := make(chan string, 5)
		results := make(chan runners.TestResult, 5)
		var wg sync.WaitGroup

		wg.Add(1)
		testcollection(mockPig, jobs, &wg)
		close(jobs) // close manually since we aren't using the goroutine wrapper here

		if len(jobs) != 1 {
			t.Errorf("Expected 1 job collected, got %d", len(jobs))
		}

		wg.Add(1)
		worker(mockPig, jobs, results, &wg)
		close(results)

		res := <-results
		if !res.Passed {
			t.Error("Expected mock worker to report a passing test status")
		}
		if res.Testname != "mock_test_case" {
			t.Errorf("Expected test name 'mock_test_case', got %s", res.Testname)
		}
	})
}

func TestRunCmd_MissingLangFlag(t *testing.T) {
	// Create a brand new local instance of the command to completely isolate flag parsing
	localRunCmd := &cobra.Command{
		Use: "run",
		Run: runCmd.Run, // reuse the exact production runner logic safely
	}

	// Re-register the required flags strictly for this test lifecycle
	var localLang string
	localRunCmd.Flags().StringVarP(&localLang, "lang", "l", "", "Language to run tests for")
	_ = localRunCmd.MarkFlagRequired("lang")

	buf := new(bytes.Buffer)
	localRunCmd.SetOut(buf)
	localRunCmd.SetErr(buf)

	// Explicitly pass empty arguments to force validation to trigger
	localRunCmd.SetArgs([]string{})
	err := localRunCmd.Execute()

	// Assert that Cobra safely caught the error before hitting your logic
	if err == nil {
		t.Fatal("Expected validation error due to missing required flag '--lang', got nil")
	}

	expectedMsg := "required flag(s)"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected flag validation warning containing %q, got: %v", expectedMsg, err)
	}
}
