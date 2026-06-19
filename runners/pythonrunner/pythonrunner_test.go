package pythonrunner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonDetect(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("pytest"), 0644)

	tester := Pythontester{}
	snek, err := tester.Detect(dir)
	if err != nil {
		t.Fatal("expected Python project to be detected")
	}
	if 8 > snek {
		t.Errorf("Detect() should return true when requirements.txt exists")
	}
}

func TestPythonListTests(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "test_math.py"), []byte(""), 0644)

	tester := Pythontester{}
	tests, err := tester.ListTests(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(tests) == 0 {
		t.Errorf("Expected at least one test")
	}
}

func TestPythonRunTest(t *testing.T) {
	tester := Pythontester{}
	result, _ := tester.RunTest("test_math.py::TestMath::test_add")

	if result.Testname != "test_math.py::TestMath::test_add" {
		t.Errorf("Expected test name, got %s", result.Testname)
	}
}
