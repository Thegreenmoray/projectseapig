package javarunner

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Justi/projectseapig/runners"
)

type Javatester struct {
}

func (g Javatester) Detect(projectPath string) bool {
	// Maven
	if _, err := os.Stat(filepath.Join(projectPath, "pom.xml")); err == nil {
		return true
	}

	// Gradle
	if _, err := os.Stat(filepath.Join(projectPath, "build.gradle")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(projectPath, "build.gradle.kts")); err == nil {
		return true
	}

	// Java test files (src/test/java)
	testPattern := filepath.Join(projectPath, "src", "test", "java", "**", "*.java")
	matches, _ := filepath.Glob(testPattern)
	if len(matches) > 0 {
		return true
	}

	// Any .java files in the project root (fallback)
	rootJava, _ := filepath.Glob(filepath.Join(projectPath, "*.java"))
	return len(rootJava) > 0
}

func (g Javatester) ListTests(projectPath string) ([]string, error) {

	// Look for Java test files
	pattern := filepath.Join(projectPath, "src", "test", "java", "**", "*Test.java")

	// If the directory doesn't exist, return empty list
	if _, err := os.Stat(pattern); os.IsNotExist(err) {
		return []string{}, nil
	}
	// NOTE: Go's Glob does NOT support **, so we walk manually
	var tests []string

	filepath.WalkDir(filepath.Join(projectPath, "src", "test", "java"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), "Test.java") {
			// Extract class name (file name without .java)
			className := strings.TrimSuffix(d.Name(), ".java")
			tests = append(tests, className)
		}

		return nil
	})

	return tests, nil

}

func (g Javatester) RunTest(testName string) (runners.TestResult, error) {
	var cmd *exec.Cmd

	// Detect Maven
	if _, err := os.Stat("pom.xml"); err == nil {
		cmd = exec.Command("mvn", "-q", "-Dtest="+testName, "test")
	}

	// Detect Gradle
	if cmd == nil {
		if _, err := os.Stat("build.gradle"); err == nil {
			cmd = exec.Command("gradle", "test", "--tests", testName)
		} else if _, err := os.Stat("build.gradle.kts"); err == nil {
			cmd = exec.Command("gradle", "test", "--tests", testName)
		}
	}

	// If no build system found
	if cmd == nil {
		return runners.TestResult{
			Testname: testName,
			Passed:   false,
			Stdout:   "No Maven or Gradle build file found",
		}, nil
	}

	// Run the command
	out, err := cmd.CombinedOutput()
	passed := err == nil

	return runners.TestResult{
		Testname: testName,
		Passed:   passed,
		Stdout:   string(out),
	}, nil

}
