package pythonrunner

import (
	"os"
	//"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPythonListTests_Integration(t *testing.T) {
	//	if _, err := exec.LookPath("pytest"); err != nil {
	//		t.Skip("Skipping integration test: 'pytest' binary not found in system PATH")
	//	}

	dir := t.TempDir()

	// Write actual Python test code so pytest has something to collect!
	pythonCode := []byte("def test_addition():\n    assert 1 + 1 == 2\n")
	_ = os.WriteFile(filepath.Join(dir, "test_math.py"), pythonCode, 0644)

	tester := Pythontester{
		BinPath:  "pytest",
		BaseArgs: []string{},
		Timeout:  5 * time.Second,
	}

	tests, err := tester.ListTests(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(tests) != 1 {
		t.Errorf("Expected exactly 1 test, got %d", len(tests))
	}

	expectedTestName := "test_math.py::test_addition"
	if len(tests) > 0 && !strings.Contains(tests[0], expectedTestName) {
		t.Errorf("Expected test name to contain %q, got %q", expectedTestName, tests[0])
	}
}

func TestPythonRunTest(t *testing.T) {
	tester := Pythontester{
		BinPath:  "pytest",
		BaseArgs: []string{},
		Timeout:  5 * time.Second,
	}

	result, _ := tester.RunTest("test_math.py::TestMath::test_add")

	if result.Testname != "test_math.py::TestMath::test_add" {
		t.Errorf("Expected test name, got %s", result.Testname)
	}
}
