package javarunner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJavaListTests(t *testing.T) {
	dir := t.TempDir()

	testDir := filepath.Join(dir, "src", "test", "java")
	os.MkdirAll(testDir, 0755)

	os.WriteFile(filepath.Join(testDir, "MathTest.java"), []byte(""), 0644)

	tester := Javatester{
		Timeout: 60 * time.Second,
	}
	tests, err := tester.ListTests(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(tests) != 1 || tests[0] != "MathTest" {
		t.Errorf("Expected MathTest, got %v", tests)
	}
}
