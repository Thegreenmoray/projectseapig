package javarunner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJavaDetect(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0644)

	tester := Javatester{}
	if !tester.Detect(dir) {
		t.Errorf("Detect() should return true when pom.xml exists")
	}
}

func TestJavaListTests(t *testing.T) {
	dir := t.TempDir()

	testDir := filepath.Join(dir, "src", "test", "java")
	os.MkdirAll(testDir, 0755)

	os.WriteFile(filepath.Join(testDir, "MathTest.java"), []byte(""), 0644)

	tester := Javatester{}
	tests, err := tester.ListTests(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(tests) != 1 || tests[0] != "MathTest" {
		t.Errorf("Expected MathTest, got %v", tests)
	}
}

func TestJavaRunTest(t *testing.T) {
	tester := Javatester{}
	result, _ := tester.RunTest("MathTest")

	if result.Testname != "MathTest" {
		t.Errorf("Expected test name MathTest, got %s", result.Testname)
	}
}
