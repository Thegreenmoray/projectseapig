package javarunner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
	// Ensure your runners import is here
)

type Javatester struct {
	BinPath     string   // e.g., "mvn" or "gradlew"
	BaseArgs    []string // e.g., []string{"test"}
	Timeout     time.Duration
	Env         []string
	ProjectPath string // Added to ensure cmd.Dir points to the right spot
}

func (g *Javatester) ListTests(projectPath string) ([]string, error) {
	var tests []string

	// Target the standard Java test source directory
	testRoot := filepath.Join(projectPath, "src", "test", "java")

	searchPath := projectPath
	if _, err := os.Stat(testRoot); err == nil {
		searchPath = testRoot
	}

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), "Test.java") {
			// Get the relative path from the test root (e.g., "org/example/FlakyTest.java")
			relPath, err := filepath.Rel(searchPath, path)
			if err != nil {
				return err
			}

			// Strip ".java" and convert directory slashes (\ or /) into package dots
			cleanPath := strings.TrimSuffix(relPath, ".java")
			fqcn := strings.ReplaceAll(cleanPath, string(os.PathSeparator), ".")

			tests = append(tests, fqcn)
		}
		return nil
	})

	return tests, err
}

func (g *Javatester) RunTest(testName string) (runners.TestResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	var bin string = g.BinPath
	var args []string

	// Helper to handle Windows vs Unix local wrapper paths
	getGradleWrapper := func() string {
		if runtime.GOOS == "windows" {
			return ".\\gradlew.bat"
		}
		return "./gradlew"
	}

	// Dynamic argument selection
	if bin == "mvn" {
		args = append(g.BaseArgs, "-q", "-Dtest="+testName)
	} else if bin == "gradle" || bin == "gradlew" || strings.Contains(bin, "gradle") {
		args = append(g.BaseArgs, "--tests", testName)
		if bin == "gradlew" {
			bin = getGradleWrapper()
		}
	} else {
		// Fallback detection using the structured ProjectPath
		pomPath := filepath.Join(g.ProjectPath, "pom.xml")
		gradlePath := filepath.Join(g.ProjectPath, "build.gradle")

		if _, err := os.Stat(pomPath); err == nil {
			bin = "mvn"
			args = []string{"test", "-q", "-Dtest=" + testName}
		} else if _, err := os.Stat(gradlePath); err == nil {
			args = []string{"test", "--tests", testName}

			if _, err := os.Stat(filepath.Join(g.ProjectPath, "gradlew")); err == nil {
				bin = getGradleWrapper()
			} else {
				bin = "gradle"
			}
		} else {
			return runners.TestResult{
				Testname: testName,
				Passed:   false,
				Stdout:   "No explicit configuration or local Maven/Gradle build file discovered.",
			}, nil
		}
	}
	var absBin string
	// If it's a local wrapper script, resolve it relative to the project directory
	if bin == "gradlew" || bin == "gradlew.bat" || strings.HasPrefix(bin, ".") {
		var err error
		absBin, err = filepath.Abs(filepath.Join(g.ProjectPath, bin))
		if err != nil {
			absBin = bin
		}
	} else {
		// If it's a global tool like "mvn" or "gradle", let the OS look it up in %PATH%
		absBin = bin
	}

	// Construct command execution
	cmd := exec.CommandContext(ctx, absBin, args...)
	cmd.Dir = g.ProjectPath // Crucial fix: Forces the OS process to spawn inside your test folder

	if len(g.Env) > 0 {
		cmd.Env = g.Env
	}

	out, err := cmd.CombinedOutput()
	passed := err == nil

	// Capture underlying OS execution errors (like file not found) for debugging
	if err != nil && len(out) == 0 {
		out = append(out, []byte("\n--- SEAPIG EXEC ERROR: "+err.Error()+" ---")...)
	}

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
