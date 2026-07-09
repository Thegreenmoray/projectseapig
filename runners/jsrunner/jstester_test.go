package jsrunner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSTesterListTests(t *testing.T) {
	dir := t.TempDir()

	// Create fake test files
	os.WriteFile(filepath.Join(dir, "math.test.js"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "utils.spec.js"), []byte(""), 0644)

	tester := JStester{}
	tests, err := tester.ListTests(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(tests) != 2 {
		t.Errorf("Expected 2 tests, got %d", len(tests))
	}
}

func TestJSTesterRunTest(t *testing.T) {
	tester := JStester{}
	result, _ := tester.RunTest("math.test.js")

	if result.Testname != "math.test.js" {
		t.Errorf("Expected test name math.test.js, got %s", result.Testname)
	}
}
