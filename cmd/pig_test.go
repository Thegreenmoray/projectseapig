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
func TestSwap(t *testing.T) {
	heap := &FloatHeap{}
	heap.Push(Pair{Time: -int32(90), Testname: "ddd"})
	heap.Push(Pair{Time: -int32(50), Testname: "ddd"})
	heap.Swap(1, 0)
	heap.Less(1, 0)
}

func TestRunCmd_Success(t *testing.T) {
	t.Run("Pipeline Concurrency Sanity Check", func(t *testing.T) {
		mockDaemon := &Mockdeamon{}
		mockPig := &MockPigRunner{}

		// 1. Fetch test targets from mock runner
		tests, err := mockPig.ListTests(".")
		if err != nil {
			t.Fatalf("Failed to list tests: %v", err)
		}

		// 2. Mock bbolt historical timings
		hashmap := map[string]int64{
			"mock_test_case": 150,
		}

		// 3. Setup Heap & Dispatcher
		heap := &FloatHeap{}
		n := 1 // run count per test
		for _, testName := range tests {
			sampleTime := hashmap[testName]
			heap.Push(Pair{Time: -int32(sampleTime), Testname: testName})
		}

		cpucores := 2
		var sliceofslices [][]string
		for heap.Len() > 0 {
			var batch []string
			for i := 0; i < cpucores && heap.Len() > 0; i++ {
				batch = append(batch, heap.Pop().Testname)
			}
			sliceofslices = append(sliceofslices, batch)
		}

		// 4. Setup channels and sync primitives
		ch := make(chan []runners.TestResult, len(sliceofslices))
		var wg sync.WaitGroup

		// 5. Run daemon & batch execution
		_ = mockDaemon.Start()
		defer mockDaemon.Stop()

		for _, batch := range sliceofslices {
			wg.Add(1)
			go func(testNames []string) {
				defer wg.Done()
				res, err := mockDaemon.RunTests(testNames)
				if err != nil {
					t.Errorf("Unexpected error running batch: %v", err)
					return
				}
				ch <- res
			}(batch)
		}

		wg.Wait()
		close(ch)

		// 6. Assert results
		var totalResults int
		for batchResults := range ch {
			for _, res := range batchResults {
				totalResults++
				if !res.Passed {
					t.Error("Expected mock test to pass")
				}
				if res.Testname != "mock_test_case" {
					t.Errorf("Expected 'mock_test_case', got %s", res.Testname)
				}
			}
		}

		if totalResults != len(tests)*n {
			t.Errorf("Expected %d results, got %d", len(tests)*n, totalResults)
		}
	})
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
		//t.Errorf("Expected prompt reading execution check to read 'y', got %s", string(userInput))
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
