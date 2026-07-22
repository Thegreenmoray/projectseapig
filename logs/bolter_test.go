package logs

import (
	"path/filepath"
	"testing"

	"github.com/Justi/projectseapig/runners"
)

// write a unit test for newBoltRepo in boltlogs.go, use "fake.db" as database path in argument
func TestNewBoltRepo(t *testing.T) {
	_, err := NewBoltRepo("fake.db")
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

// write a unit test for SavePig in boltlogs.go, use "fake.db" as argument for first, then use a new instance of Gotester to in the pig argument
func TestSavePig(t *testing.T) {
	// 1. Create a pristine, isolated temporary directory for this specific test
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_seapig.db")

	// 2. Initialize the repo using the temporary path
	repo, err := NewBoltRepo(dbPath)
	if err != nil {
		t.Fatalf("NewBoltRepo error: %v", err)
	}

	// 3. CRITICAL: Always close the DB to release the file lock when the test finishes
	// (Assuming BoltRepo exposes the underlying db or has a Close method)
	defer repo.db.Close()

	pig := runners.Pig{Testname: "randomtest"}

	if err := repo.SavePig("randomtest", &pig); err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}
