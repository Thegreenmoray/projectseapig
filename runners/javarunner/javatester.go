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

	bin := g.BinPath
	var args []string

	// 1. Build CLI args with performance flags (-q for quiet, -o for offline/no-remote-check)
	if strings.Contains(bin, "mvn") {
		args = append([]string{"test", "-q", "-o", "-B", "-Dtest=" + testName})
	} else {
		// Gradle execution
		args = append([]string{"test", "-q", "--tests", testName})
		if bin == "gradlew" {
			if runtime.GOOS == "windows" {
				bin = ".\\gradlew.bat"
			} else {
				bin = "./gradlew"
			}
		}
	}

	// 2. Resolve relative path for wrapper scripts
	absBin := bin
	if strings.HasPrefix(bin, ".") || bin == "gradlew.bat" {
		if resolved, err := filepath.Abs(filepath.Join(g.ProjectPath, bin)); err == nil {
			absBin = resolved
		}
	}

	// 3. Command setup
	cmd := exec.CommandContext(ctx, absBin, args...)
	cmd.Dir = g.ProjectPath

	if len(g.Env) > 0 {
		cmd.Env = g.Env
	}

	start := time.Now()
	out, err := cmd.CombinedOutput()

	return runners.TestResult{
		Testname:  testName,
		Passed:    err == nil,
		Stdout:    string(out),
		Timetaken: time.Since(start),
	}, nil
}
