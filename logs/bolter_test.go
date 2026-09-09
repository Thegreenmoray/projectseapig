package logs

import (
	"path/filepath"
	"testing"
	"time"

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
func setupTestRepo(t *testing.T) *BoltRepo {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test_sea_pig.db")
	repo, err := NewBoltRepo(dbPath)
	if err != nil {
		t.Fatalf("Failed to create temp BoltRepo: %v", err)
	}
	return repo
}

func TestBoltRepo_SavePig_And_Close(t *testing.T) {
	repo := setupTestRepo(t)

	pig := &runners.Pig{
		Dateandtime: time.Now().Format(time.RFC3339),
		Run: []runners.TestResult{
			{Testname: "TestAuthLogin", Passed: true, Timetaken: 150 * time.Millisecond},
		},
	}

	// 1. Test SavePig
	if err := repo.SavePig("TestAuthLogin", pig); err != nil {
		t.Fatalf("SavePig() failed: %v", err)
	}

	// 2. Test Close
	if err := repo.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Verify idempotency on nil/closed DB
	var emptyRepo BoltRepo
	if err := emptyRepo.Close(); err != nil {
		t.Errorf("Expected nil error when closing uninitialized repo, got: %v", err)
	}
}

func TestBoltRepo_SavePigtime_And_Extractpigtime(t *testing.T) {
	repo := setupTestRepo(t)
	defer repo.Close()

	// 1. Test Extractpigtime before bucket exists (Error path)
	_, err := repo.Extractpigtime()
	if err == nil {
		t.Error("Expected error when extracting from non-existent TestTime bucket, got nil")
	}

	pig := &runners.Pig{
		Run: []runners.TestResult{
			{Testname: "TestDBConnect", Passed: true, Timetaken: 200 * time.Millisecond},
			{Testname: "TestDBClose", Passed: true, Timetaken: 50 * time.Millisecond},
		},
	}

	// 2. Test SavePigtime
	testName := "SuiteDatabase"
	if err := repo.SavePigtime(testName, pig); err != nil {
		t.Fatalf("SavePigtime() failed: %v", err)
	}

	// 3. Test Extractpigtime success path
	timesMap, err := repo.Extractpigtime()
	if err != nil {
		t.Fatalf("Extractpigtime() failed unexpectedly: %v", err)
	}

	expectedKey0 := "SuiteDatabase_0"
	expectedKey1 := "SuiteDatabase_1"

	if val, ok := timesMap[expectedKey0]; !ok || val != int((200*time.Millisecond).Nanoseconds()) {
		t.Errorf("Expected %d ns for key %s, got %d", (200 * time.Millisecond).Nanoseconds(), expectedKey0, val)
	}

	if val, ok := timesMap[expectedKey1]; !ok || val != int((50*time.Millisecond).Nanoseconds()) {
		t.Errorf("Expected %d ns for key %s, got %d", (50 * time.Millisecond).Nanoseconds(), expectedKey1, val)
	}
}
