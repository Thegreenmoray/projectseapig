package gorunner

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Justi/projectseapig/runners"
)

type Gotester struct {
}

func (g *Gotester) Detect(projectPath string) (int, error) {
	score := 0

	err := dfsWalk(projectPath, &score)
	if err != nil {
		return 0, nil
	}

	return score, nil
}

func (g *Gotester) ListTests(projectPath string) ([]string, error) {
	//basic command line
	cmd := exec.Command("go", "test", "-list", ".", projectPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var tests []string
	//equventanet to an enhanced for loop
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Test") {
			tests = append(tests, line)
		}
	}

	return tests, nil
}

func (g *Gotester) RunTest(testName string) (runners.TestResult, error) {

	cmd := exec.Command("go", "test", "-run", "^"+testName+"$")
	out, err := cmd.CombinedOutput()

	passed := err == nil

	return runners.TestResult{
		Testname: testName,
		Passed:   passed,
		Stdout:   string(out),
	}, nil
}

func dfsWalk(path string, score *int) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Strongest signal: go.mod
	if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
		*score += 10
	}

	for _, entry := range entries {
		full := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			// Recurse
			if err := dfsWalk(full, score); err != nil {
				return err
			}
			continue
		}

		// --- FILE CHECKS ---

		// 1. Test file pattern
		if strings.HasSuffix(entry.Name(), "_test.go") {
			*score += 5
		}

		// 2. Only scan .go files
		if strings.HasSuffix(entry.Name(), ".go") {
			if err := scanGoFile(full, score); err != nil {
				return err
			}
		}
	}

	return nil
}

func scanGoFile(path string, score *int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()

		// Trim whitespace
		line = strings.TrimSpace(line)

		// Go language signatures
		switch {
		case strings.HasPrefix(line, "package "):
			*score += 3
		case strings.HasPrefix(line, "import "):
			*score += 2
		case strings.HasPrefix(line, "func "):
			*score += 2
		case strings.Contains(line, "struct {"):
			*score += 1
		case strings.Contains(line, "interface {"):
			*score += 1
		case strings.Contains(line, "go "): // goroutine
			*score += 1
		case strings.Contains(line, "chan "):
			*score += 1
		case strings.HasPrefix(line, "//go:build"):
			*score += 2
		}

		// Early exit: if score is already high, no need to scan whole file
		if *score >= 15 {
			break
		}
	}

	return scanner.Err()
}
