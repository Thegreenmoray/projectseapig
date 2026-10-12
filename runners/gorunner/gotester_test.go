package gorunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGotester_ListTests_Coverage(t *testing.T) {
	// Helper temp files/dirs
	tempDir := t.TempDir()

	// Create a dummy file to test non-directory validation
	dummyFilePath := filepath.Join(tempDir, "not_a_dir.txt")
	if err := os.WriteFile(dummyFilePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create dummy file: %v", err)
	}

	// Create a directory with a valid Go test file
	validGoDir := filepath.Join(tempDir, "validpkg")
	if err := os.MkdirAll(validGoDir, 0755); err != nil {
		t.Fatalf("failed to create validpkg dir: %v", err)
	}
	goTestContent := `package validpkg
import "testing"
func TestSampleOne(t *testing.T) {}
func TestSampleTwo(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(validGoDir, "sample_test.go"), []byte(goTestContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create a directory with NO Go tests to hit the empty result error branch
	emptyGoDir := filepath.Join(tempDir, "emptypkg")
	if err := os.MkdirAll(emptyGoDir, 0755); err != nil {
		t.Fatalf("failed to create emptypkg dir: %v", err)
	}

	tests := []struct {
		name        string
		tester      Gotester
		path        string
		expectErr   bool
		minExpected int
	}{
		{
			name:      "Error on non-positive timeout",
			tester:    Gotester{Timeout: 0},
			path:      ".",
			expectErr: true,
		},
		{
			name:      "Error on non-existent path",
			tester:    Gotester{Timeout: 5 * time.Second},
			path:      filepath.Join(tempDir, "does_not_exist_12345"),
			expectErr: true,
		},
		{
			name:      "Error on path that is a file, not a directory",
			tester:    Gotester{Timeout: 5 * time.Second},
			path:      dummyFilePath,
			expectErr: true,
		},
		{
			name:        "Default BinPath fallback to 'go'",
			tester:      Gotester{BinPath: "", Timeout: 10 * time.Second},
			path:        validGoDir,
			expectErr:   true,
			minExpected: 2,
		},
		{
			name:      "Error when no tests found in package",
			tester:    Gotester{BinPath: "go", Timeout: 5 * time.Second},
			path:      emptyGoDir,
			expectErr: true,
		},
		{
			name:        "Success listing project root",
			tester:      Gotester{BinPath: "go", Timeout: 10 * time.Second},
			path:        ".",
			expectErr:   false,
			minExpected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.tester.ListTests(tt.path)
			if (err != nil) != tt.expectErr {
				t.Fatalf("ListTests() error = %v, expectErr %v", err, tt.expectErr)
			}
			if tt.name == "Error when no tests found in package" && !strings.Contains(err.Error(), "check that --lang matches this project") {
				t.Errorf("expected actionable language hint, got %v", err)
			}
			if !tt.expectErr && len(got) < tt.minExpected {
				t.Errorf("Expected at least %d tests, got %d", tt.minExpected, len(got))
			}
		})
	}
}

func TestGotester_ListTestsReportsPackageProgress(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/progress\n\ngo 1.21\n"), 0600); err != nil {
		t.Fatal(err)
	}
	rootTest := "package progress\nimport \"testing\"\nfunc TestRoot(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(projectDir, "root_test.go"), []byte(rootTest), 0600); err != nil {
		t.Fatal(err)
	}

	subpackageDir := filepath.Join(projectDir, "subpackage")
	if err := os.MkdirAll(subpackageDir, 0755); err != nil {
		t.Fatal(err)
	}
	subpackageTest := "package subpackage\nimport \"testing\"\nfunc TestSubpackage(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(subpackageDir, "subpackage_test.go"), []byte(subpackageTest), 0600); err != nil {
		t.Fatal(err)
	}

	lastCompleted, lastTotal := 0, 0
	seenPackage := make(map[string]bool)
	tester := Gotester{
		BinPath:          "go",
		Timeout:          time.Minute,
		DiscoveryTimeout: time.Minute,
		DiscoveryProgress: func(completed, total int, packagePath string) {
			lastCompleted, lastTotal = completed, total
			if packagePath != "" {
				seenPackage[packagePath] = true
			}
		},
	}
	tests, err := tester.ListTests(projectDir)
	if err != nil {
		t.Fatalf("ListTests() error = %v", err)
	}
	if len(tests) != 2 {
		t.Fatalf("ListTests() found %d tests, want 2: %v", len(tests), tests)
	}
	if lastTotal != 2 || lastCompleted != lastTotal {
		t.Fatalf("final discovery progress = %d/%d, want 2/2", lastCompleted, lastTotal)
	}
	if len(seenPackage) != 2 {
		t.Fatalf("reported %d packages, want both discovered packages: %v", len(seenPackage), seenPackage)
	}
}
