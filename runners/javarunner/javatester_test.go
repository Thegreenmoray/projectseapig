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

func TestJavaRunTestpom(t *testing.T) {
	tester := Javatester{
		Timeout:  60 * time.Second,
		BinPath:  "mvn",
		BaseArgs: []string{"test"},
	}
	dir := t.TempDir()

	// 1. Create pom.xml inside the temp directory
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0644)

	// 2. Save the original working directory path
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	// 3. Jump inside the temp directory
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	// 4. CRITICAL: Tell Go to jump back to your project folder when this test finishes
	defer os.Chdir(oldWD)

	// 5. Run the test! Now os.Stat("pom.xml") evaluates to true!
	result, _ := tester.RunTest("MathTest")

	if result.Testname != "MathTest" {
		t.Errorf("Expected test name MathTest, got %s", result.Testname)
	}
}

func TestJavaRunTestgradle(t *testing.T) {
	tester := Javatester{
		Timeout:  60 * time.Second,
		BinPath:  "gradlew.bat",
		BaseArgs: []string{"test"},
	}
	dir := t.TempDir()

	// 1. Create build.gradle inside the temp directory
	os.WriteFile(filepath.Join(dir, "build.gradle"), []byte("<project/>"), 0644)

	// 2. Save the original working directory path
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	// 3. Jump inside the temp directory
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	// 4. CRITICAL: Tell Go to jump back to your project folder when this test finishes
	defer os.Chdir(oldWD)

	// 5. Run the test! Now os.Stat("build.gradle") evaluates to true!
	result, _ := tester.RunTest("MathTest")

	if result.Testname != "MathTest" {
		t.Errorf("Expected test name MathTest, got %s", result.Testname)
	}

}
