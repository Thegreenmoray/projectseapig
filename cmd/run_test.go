package cmd

import (
	"bytes"
	"strings"

	"testing"

	"github.com/Justi/projectseapig/runners"
	"github.com/spf13/cobra"
)

// 1. Create a lightweight mock that satisfies the runners.TestRunner interface
type MockPigRunner struct {
	ShouldFailExecution bool
}

type Mockdeamon struct {
	ShouldFailExecution bool
}

func (e *Mockdeamon) Start() error {

	return nil
}
func (e *Mockdeamon) Stop() error {

	return nil
}

func (e *Mockdeamon) RunTests(names []string) ([]runners.TestResult, error) {
	var results []runners.TestResult
	for _, name := range names {
		results = append(results, runners.TestResult{
			Testname: name,
			Passed:   true,
		})
	}
	return results, nil
}

func (m *MockPigRunner) ListTests(projectPath string) ([]string, error) {
	// Return a single predictable test target name
	return []string{"mock_test_case"}, nil
}

// 2. The Unit Test Suite

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
