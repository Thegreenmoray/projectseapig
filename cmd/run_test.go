package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Justi/projectseapig/runners"
	"github.com/spf13/cobra"
)

// 1. Create a lightweight mock that satisfies the runners.TestRunner interface
type MockPigRunner struct {
	ShouldFailExecution bool
}

func TestListTestsWithProgressReturnsDiscoveryResults(t *testing.T) {
	want := []string{"pkg::TestA", "pkg::TestB"}
	got, err := listTestsWithProgress(func(reportProgress func(completed, total int, packagePath string)) ([]string, error) {
		reportProgress(0, 2, "")
		reportProgress(0, 2, "pkg/a")
		reportProgress(1, 2, "pkg/a")
		reportProgress(1, 2, "pkg/b")
		reportProgress(2, 2, "pkg/b")
		return want, nil
	})
	if err != nil {
		t.Fatalf("listTestsWithProgress() error = %v", err)
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("listTestsWithProgress() = %v, want %v", got, want)
	}
}

func TestRenderPackageProgress(t *testing.T) {
	var output bytes.Buffer
	renderPackageProgress(&output, 1, 4)
	if got, want := output.String(), "\r[seapig] Scanning Go packages: [=====---------------] 1/4"; got != want {
		t.Fatalf("renderPackageProgress() = %q, want %q", got, want)
	}
}

func TestRunCmd_DiscoveryErrorIsFailure(t *testing.T) {
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previousDirectory); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	}()

	projectDirectory := t.TempDir()
	module := []byte("module example.com/no-tests\n\ngo 1.21\n")
	if err := os.WriteFile(filepath.Join(projectDirectory, "go.mod"), module, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDirectory, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDirectory); err != nil {
		t.Fatal(err)
	}

	previousLanguage := lang
	lang = "go"
	defer func() { lang = previousLanguage }()

	err = runCmd.RunE(runCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to discover tests") {
		t.Fatalf("run command error = %v, want a test-discovery failure", err)
	}
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
		Use:  "run",
		RunE: runCmd.RunE, // reuse the exact production runner logic safely
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
