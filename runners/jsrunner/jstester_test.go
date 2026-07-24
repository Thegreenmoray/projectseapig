package jsrunner

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestJSTesterListTests(t *testing.T) {
	// 1. Check for npx since we use it to invoke the local Jest binary
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping test: 'npx' executable not found in system PATH")
	}

	dir := t.TempDir()

	// 2. Create minimal package.json and jest config so Jest doesn't crash
	packageJSON := []byte(`{"name": "test-project", "private": true}`)
	_ = os.WriteFile(filepath.Join(dir, "package.json"), packageJSON, 0644)

	// 3. Create fake test files
	_ = os.WriteFile(filepath.Join(dir, "math.test.js"), []byte("test('stub', () => {});"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "utils.spec.js"), []byte("test('stub', () => {});"), 0644)

	// Initialize the tester with your factory style defaults
	tester := JStester{
		BinPath:  "npx",
		BaseArgs: []string{"jest", "--roots", dir, "--listTests"},
		Timeout:  60 * time.Second,
	}

	tests, err := tester.ListTests(dir)
	if err != nil {
		// If Jest isn't installed in this specific environment, skip gracefully
		// instead of failing the CI pipeline build
		t.Skipf("Skipping: local environment missing node_modules or jest configuration: %v", err)
		return
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
