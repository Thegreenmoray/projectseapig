package jsrunner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSTesterDetect(t *testing.T) {
	dir := t.TempDir()

	// Create a fake package.json
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	tester := JStester{}
	addo, err := tester.Detect(dir)
	if err != nil {
		t.Fatal("expected JS project to be detected")
	}
	if addo < 10 {
		t.Errorf("Detect() should return true when package.json exists")
	}
}

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
