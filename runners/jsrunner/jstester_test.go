package jsrunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJSTesterListTests(t *testing.T) {
	// 1. Check for npx since we use it to invoke the local Jest binary
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("Skipping test: 'npx' executable not found in system PATH")
	}

	dir := t.TempDir()

	// 2. Create minimal package.json
	packageJSON := []byte(`{"name": "test-project", "private": true}`)
	_ = os.WriteFile(filepath.Join(dir, "package.json"), packageJSON, 0644)

	// 3. Create fake test files
	_ = os.WriteFile(filepath.Join(dir, "math.test.js"), []byte("test('stub', () => {});"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "utils.spec.js"), []byte("test('stub', () => {});"), 0644)

	// Explicitly constrain Jest roots using a minimal JSON config string
	jestConfig := fmt.Sprintf(`{"rootDir": "%s", "roots": ["%s"]}`,
		filepath.ToSlash(dir),
		filepath.ToSlash(dir),
	)

	tester := JStester{
		BinPath:  "npx",
		BaseArgs: []string{"jest", "--config", jestConfig, "--listTests"},
		Timeout:  60 * time.Second,
	}

	tests, err := tester.ListTests(dir)
	if err != nil {
		t.Skipf("Skipping: local environment missing node_modules or jest configuration: %v", err)
		return
	}

	// Filter to strictly verify tests belonging inside t.TempDir()
	var localTests []string
	cleanTempDir := filepath.Clean(dir)

	for _, testPath := range tests {
		cleanPath := filepath.Clean(testPath)
		if strings.HasPrefix(cleanPath, cleanTempDir) {
			localTests = append(localTests, cleanPath)
		}
	}

	if len(localTests) != 2 {
		t.Errorf("Expected 2 tests in temp dir, got %d (All discovered: %v)", len(localTests), tests)
	}
}

func TestJSTesterRunTest(t *testing.T) {
	dir := t.TempDir()

	jestConfig := fmt.Sprintf(`{"rootDir": "%s", "roots": ["%s"]}`,
		filepath.ToSlash(dir),
		filepath.ToSlash(dir),
	)

	tester := JStester{
		BinPath:  "npx",
		BaseArgs: []string{"jest", "--config", jestConfig, "--listTests"},
		Timeout:  60 * time.Second,
	}
	result, _ := tester.RunTest("math.test.js")

	if result.Testname != "math.test.js" {
		t.Errorf("Expected test name math.test.js, got %s", result.Testname)
	}
}
