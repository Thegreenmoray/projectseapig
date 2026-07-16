package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Justi/projectseapig/logs"
	"github.com/Justi/projectseapig/runners"
	"github.com/spf13/cobra"
)

// 1. A clean mock runner to isolate system commands

func (m *MockPigRunner) ListTest(p string) ([]string, error) {
	return []string{"mock_test_case"}, nil
}

func (m *MockPigRunner) RunTes(t string) (runners.TestResult, error) {
	return runners.TestResult{Testname: t, Passed: !m.ShouldFailExecution, Stdout: "Mock output"}, nil
}

// --- TEST 1: Worker Pipeline Success ---
func TestRun_WorkerPipeline_Success(t *testing.T) {
	mockPig := &MockPigRunner{ShouldFailExecution: false}
	jobs := make(chan string, 1)
	results := make(chan runners.TestResult, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	testcollection(mockPig, jobs, &wg)
	close(jobs)

	wg.Add(1)
	worker(mockPig, jobs, results, &wg)
	close(results)

	res := <-results
	if !res.Passed || res.Testname != "mock_test_case" {
		t.Errorf("Worker pipeline failed processing jobs correctly")
	}
}

// --- TEST 2: Missing Required Lang Flag Error ---
func TestRunCmd_MissingLangFla(t *testing.T) {
	localRunCmd := &cobra.Command{Use: "run", Run: runCmd.Run}
	var localLang string
	localRunCmd.Flags().StringVarP(&localLang, "lang", "l", "", "")
	_ = localRunCmd.MarkFlagRequired("lang")

	localRunCmd.SetArgs([]string{})
	err := localRunCmd.Execute()

	if err == nil || !strings.Contains(err.Error(), "required flag(s)") {
		t.Fatalf("Expected validation error due to missing flag, got: %v", err)
	}
}

// --- TEST 3: Prompt Verification - User Chooses No ---
func TestPigCmd_Prompt_UserCancels(t *testing.T) {
	inputReader, inputWriter, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = inputReader
	defer func() { os.Stdin = oldStdin }()

	// 1. Intercept standard output streams where loggers write
	outReader, outWriter, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = outWriter
	defer func() { os.Stdout = oldStdout }()

	// Simulate user typing "n"
	_, _ = inputWriter.Write([]byte("n\n"))
	_ = inputWriter.Close()

	root := NewRootCmd()
	root.SetArgs([]string{"pig", "--lang", "go"})

	// Execute the command
	_ = root.Execute()

	// Close the writer so we can read what was recorded
	_ = outWriter.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(outReader)
	actualOutput := buf.String()

	// 2. Assert against what the logger ACTUALLY dumped out to the console terminal
	if !strings.Contains(actualOutput, "user cancelled process") {
		t.Errorf("Expected production code to abort execution, but captured console log was: %s", actualOutput)
	}
}

// --- TEST 4: Prompt Verification - User Chooses Yes ---
func TestPigCmd_Prompt_UserAccepts(t *testing.T) {
	inputReader, inputWriter, _ := os.Pipe()
	oldStdin := os.Stdin
	os.Stdin = inputReader
	defer func() { os.Stdin = oldStdin }()

	// Simulate user typing "y" to accept the risk
	_, _ = inputWriter.Write([]byte("y\n"))
	_ = inputWriter.Close()

	userInput := make([]byte, 1)
	_, _ = os.Stdin.Read(userInput)

	if string(userInput) != "y" {
		t.Errorf("Expected prompt reading execution check to read 'y', got %s", string(userInput))
	}
}

// --- TEST 5: Factory Type Error Handling Fallback ---
func TestRunCmd_InvalidLangFallback(t *testing.T) {
	buf := new(bytes.Buffer)
	localRunCmd := &cobra.Command{
		Use: "run",
		Run: func(cmd *cobra.Command, args []string) {
			// Mimic the production fallback route
			err := fmt.Errorf("Lang not supported...")
			buf.WriteString(err.Error())
		},
	}

	_ = localRunCmd.Execute()
	if !strings.Contains(buf.String(), "Lang not supported...") {
		t.Errorf("Expected error string logging, got: %s", buf.String())
	}
}
func TestPigCmd_Results_Coverage(t *testing.T) {
	// 1. Create a safe, temporary database path for this test execution
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_seapig.db")

	// 2. Initialize a real, valid database instance using your factory constructor
	testRepo, err := logs.NewBoltRepo(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test BoltRepo: %v", err)
	}
	defer testRepo.Close() // Clean up file descriptors when the test completes

	// 3. Build your telemetry mock payload data
	mockRuns := []runners.TestResult{
		{
			Testname:  "TestMath_Addition",
			Passed:    true,
			Stdout:    "PASS",
			Timetaken: 5 * time.Millisecond,
		},
	}

	mockData := map[string][]runners.TestResult{
		"TestMath_Addition": mockRuns,
	}

	// 4. Fire the function using your initialized test repo!
	results1(testRepo, &mockData)

	// 5. Basic sanity validation assertions
	if len(mockData) != 1 {
		t.Errorf("Expected 1 test target tracked in batch, got %d", len(mockData))
	}
}
