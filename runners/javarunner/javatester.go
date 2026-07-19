package javarunner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type Javatester struct {
	BinPath  string        // e.g., "/usr/local/go/bin/go" or just "go"
	BaseArgs []string      // e.g., []string{"test", "-run"}
	Timeout  time.Duration // Individual test execution timeout
	Env      []string      // Custom ENV vars for the test runner process
}

func (g *Javatester) ListTests(projectPath string) ([]string, error) {
	var tests []string

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), "Test.java") {
			name := strings.TrimSuffix(info.Name(), ".java")
			tests = append(tests, name)
		}
		return nil
	})

	return tests, err

}

func (g *Javatester) RunTest(testName string) (runners.TestResult, error) {
	// 1. Initialize context using our struct's Timeout value
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	var bin string = g.BinPath
	var args []string

	// 2. Build arguments dynamically based on whether we are using Maven or Gradle
	// If the user specified 'mvn' or it's the factory default
	if bin == "mvn" {
		args = append(g.BaseArgs, "-q", "-Dtest="+testName)
	} else if bin == "gradle" || bin == "gradlew" || strings.Contains(bin, "gradle") {
		// Prepend a wildcard so Gradle finds the class regardless of its package package
		args = append(g.BaseArgs, "--tests", "*."+testName)
	} else {
		// Fallback: If BinPath is generic, check the local directory to infer the tool
		if _, err := os.Stat("pom.xml"); err == nil {
			bin = "mvn"
			args = []string{"test", "-q", "-Dtest=" + testName}
		} else if _, err := os.Stat("build.gradle"); err == nil || os.IsExist(err) {
			bin = "gradle"
			args = []string{"test", "--tests", testName}
		} else {
			return runners.TestResult{
				Testname: testName,
				Passed:   false,
				Stdout:   "No explicit configuration or local Maven/Gradle build file discovered.",
			}, nil
		}
	}

	// 3. Construct the execution block with the Timeout context
	cmd := exec.CommandContext(ctx, bin, args...)
	if len(g.Env) > 0 {
		cmd.Env = g.Env
	}

	out, err := cmd.CombinedOutput()
	passed := err == nil

	// 4. Handle process termination if the context hits its limit
	if ctx.Err() == context.DeadlineExceeded {
		passed = false
		out = append(out, []byte("\n--- PROJECT SEAPIG: Java execution timed out! ---")...)
	}

	return runners.TestResult{
		Testname: testName,
		Passed:   passed,
		Stdout:   string(out),
	}, nil
}
