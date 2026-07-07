package javarunner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Justi/projectseapig/runners"
)

type Javatester struct {
}

func (g *Javatester) Detect(projectPath string) (int, error) {
	score := 0

	return score, nil
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
