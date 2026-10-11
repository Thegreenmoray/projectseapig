package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	}, "go")
	if err != nil {
		t.Fatalf("listTestsWithProgress() error = %v", err)
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("listTestsWithProgress() = %v, want %v", got, want)
	}
}

func TestRenderDiscoveryProgress(t *testing.T) {
	for _, test := range []struct {
		name     string
		progress discoveryProgress
		label    string
		elapsed  time.Duration
		frame    int
		want     string
	}{
		{"Go packages", discoveryProgress{completed: 1, total: 4, item: "pkg/a"}, "Go packages", 2 * time.Second, 0, "\r\033[K[seapig] Scanning Go packages: [=====---------------] 1/4 | a | 2s"},
		{"Java tests", discoveryProgress{completed: 1, total: 4, item: "src/test/java/WidgetTest.java"}, "Java tests", 2 * time.Second, 0, "\r\033[K[seapig] Scanning Java tests: [=====---------------] 1/4 | WidgetTest.java | 2s"},
		{"Python tests in progress", discoveryProgress{completed: 3, total: -1, item: "test_math.py::test_addition"}, "Python tests", 2 * time.Second, 0, "\r\033[K[seapig] Scanning Python tests: [====----------------] 3 test targets found so far | test_math.py::test_addition | 2s"},
		{"Python discovery waiting", discoveryProgress{completed: 0, total: -1, item: "waiting for pytest collection"}, "Python tests", 5 * time.Second, 5, "\r\033[K[seapig] Scanning Python tests: [-----====-----------] 0 test targets found so far | waiting for pytest collection | 5s"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			renderDiscoveryProgress(&output, test.progress, test.label, test.elapsed, test.frame)
			if got := output.String(); got != test.want {
				t.Fatalf("renderDiscoveryProgress() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestTestDiscoveryLabelUsesSelectedLanguage(t *testing.T) {
	tests := map[string]string{
		"go":     "Go packages",
		"java":   "Java tests",
		"kotlin": "Kotlin tests",
		"js":     "JavaScript tests",
		"ts":     "TypeScript tests",
		"python": "Python tests",
	}
	for language, want := range tests {
		if got := testDiscoveryLabel(language); got != want {
			t.Errorf("testDiscoveryLabel(%q) = %q, want %q", language, got, want)
		}
	}
}

func TestMatchingTestSelectorAggregatesFrameworkResults(t *testing.T) {
	selectors := []string{"com.example.WidgetTest", "src/widget.test.ts"}
	tests := []struct {
		resultName string
		want       string
		wantOK     bool
	}{
		{"com.example.WidgetTest::handles empty value", "com.example.WidgetTest", true},
		{"src/widget.test.ts::renders widget", "src/widget.test.ts", true},
		{"com.example.WidgetTest", "com.example.WidgetTest", true},
		{"<daemon>", "", false},
	}
	for _, test := range tests {
		got, ok := matchingTestSelector(test.resultName, selectors)
		if got != test.want || ok != test.wantOK {
			t.Errorf("matchingTestSelector(%q) = (%q, %t), want (%q, %t)", test.resultName, got, ok, test.want, test.wantOK)
		}
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
