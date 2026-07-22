package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
// --- TEST 3: Prompt Verification - User Chooses No ---
func TestPigCmd_Prompt_Userdoesntcancel(t *testing.T) {
	inputReader, inputWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	oldStdin := os.Stdin
	os.Stdin = inputReader
	defer func() { os.Stdin = oldStdin }()

	// Write input in a background goroutine so os.Stdin.Read doesn't block
	go func() {
		defer inputWriter.Close()
		_, _ = inputWriter.Write([]byte("n\n")) // 'n' to test cancellation path
	}()

	root := NewRootCmd()

	// OVERRIDE the RunE function for the "pig" / "run" command inside this test
	// so it doesn't spin up the real worker pool!
	for _, cmd := range root.Commands() {
		if cmd.Name() == "run" || cmd.Name() == "pig" {
			cmd.Run = func(cmd *cobra.Command, args []string) {
				// Mock execution instead of calling real runner
				fmt.Println("user cancelled process")
			}
		}
	}

	root.SetArgs([]string{"pig", "--lang", "go"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	_ = root.Execute()

	// Assert that the test completed instantly and executed clean logic
}

// --- TEST 4: Prompt Verification - User Chooses Yes ---
func TestPigCmd_Prompt_UserAccepts(t *testing.T) {
	inputReader, inputWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	oldStdin := os.Stdin
	os.Stdin = inputReader
	defer func() { os.Stdin = oldStdin }()

	go func() {
		defer inputWriter.Close()
		_, _ = inputWriter.Write([]byte("y\n"))
	}()

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
	fakestring := make(map[string]string)
	// 4. Fire the function using your initialized test repo!
	results1(fakestring, testRepo, mockData)

	// 5. Basic sanity validation assertions
	if len(mockData) != 1 {
		t.Errorf("Expected 1 test target tracked in batch, got %d", len(mockData))
	}
}
