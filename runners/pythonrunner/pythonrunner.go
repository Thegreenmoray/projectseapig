package pythonrunner

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type Pythontester struct {
	ProjectPath       string // Target workspace path (e.g., "C:\Users\...\Testsinpython")
	BinPath           string // e.g., "pytest" or path to virtualenv pytest
	BaseArgs          []string
	Timeout           time.Duration
	Env               []string
	DiscoveryProgress runners.DiscoveryProgress
}

func (g *Pythontester) SetDiscoveryProgress(progress runners.DiscoveryProgress) {
	g.DiscoveryProgress = progress
}

func (g *Pythontester) ListTests(projectPath string) ([]string, error) {
	if g.Timeout <= 0 {
		return nil, fmt.Errorf("Time is too short, please enter something larger than 0")
	}

	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	bin := g.BinPath
	if bin == "" {
		bin = "pytest"
	}

	projectPath = filepath.Clean(projectPath)
	info, err := os.Stat(projectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project path does not exist: %s", projectPath)
		}
		return nil, fmt.Errorf("error accessing project path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project path is not a directory: %s", projectPath)
	}

	args := append([]string(nil), g.BaseArgs...)
	if bin == "pytest" {
		for _, candidate := range []string{
			filepath.Join(projectPath, ".venv", "Scripts", "python.exe"),
			filepath.Join(projectPath, "venv", "Scripts", "python.exe"),
			filepath.Join(projectPath, ".venv", "bin", "python"),
			filepath.Join(projectPath, "venv", "bin", "python"),
		} {
			if candidateInfo, statErr := os.Stat(candidate); statErr == nil && !candidateInfo.IsDir() {
				bin = candidate
				args = append([]string{"-m", "pytest"}, args...)
				break
			}
		}
	}

	// --collect-only finds all tests. -q (quiet) strips unnecessary headers.
	args = append(args, "--collect-only", "-q")
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = projectPath
	if len(g.Env) > 0 {
		cmd.Env = g.Env
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("python test discovery failed: could not read command output: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		out := stderr.String()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("test discovery timed out after %v. A Python test file may be executing heavy code or network calls at module import time instead of inside a fixture", g.Timeout)
		}
		return nil, fmt.Errorf("python test discovery failed: %v | output: %s", err, out)
	}

	var output strings.Builder
	var tests []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line)
		output.WriteByte('\n')
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "no tests ran") && strings.Contains(line, "::") {
			tests = append(tests, line)
			if g.DiscoveryProgress != nil {
				g.DiscoveryProgress(len(tests), -1, line)
			}
		}
	}
	scanErr := scanner.Err()
	waitErr := cmd.Wait()
	combinedOutput := output.String() + stderr.String()
	if waitErr != nil {
		// 1. Check specifically for context timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("test discovery timed out after %v. A Python test file may be executing heavy code or network calls at module import time instead of inside a fixture", g.Timeout)
		}

		// 2. Fall back to standard command failure (syntax error, missing pytest, etc.)
		return nil, fmt.Errorf("python test discovery failed: %v | output: %s", waitErr, combinedOutput)
	}
	if scanErr != nil {
		return nil, fmt.Errorf("python test discovery failed: could not read command output: %w | output: %s", scanErr, combinedOutput)
	}

	return tests, nil
}
