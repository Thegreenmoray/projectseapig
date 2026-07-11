package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/Justi/projectseapig/logs"
	"github.com/Justi/projectseapig/runners"
)

func TestLogsCommand_Test(t *testing.T) {
	tmpDir := t.TempDir()

	// Move into temp dir so the command looks for seapig.db here
	oldWD, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer os.Chdir(oldWD)

	// 1. Seed the fake Bbolt DB with data so showSummary and showDetailedLogs both run
	repo, err := logs.NewBoltRepo("seapig.db")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	mockPig := runners.Pig{
		Testname:      "TestMath",
		PassCount:     1,
		FailCount:     1,
		Flakynessrate: 50.0,
		Run: []runners.TestResult{
			{Testname: "TestMath", Passed: true},
			{Testname: "TestMath", Passed: false, Stdout: "panic!"},
		},
	}
	_ = repo.SavePig("TestMath", mockPig)
	repo.Close() // Release lock so cmd can read it read-only

	// 2. Test 'seapig logs' (Runs showSummary)
	t.Run("shows summary", func(t *testing.T) {
		outputBuf := &bytes.Buffer{}
		root := NewRootCmd()
		root.SetOut(outputBuf)
		root.SetArgs([]string{"logs"})

		if err := root.Execute(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// 3. Test 'seapig logs TestMath' (Runs showDetailedLogs)
	t.Run("shows detailed logs", func(t *testing.T) {
		outputBuf := &bytes.Buffer{}
		root := NewRootCmd()
		root.SetOut(outputBuf)
		root.SetArgs([]string{"logs", "TestMath"})

		if err := root.Execute(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
