package jsrunner

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type JStester struct {
	ProjectPath       string // Target workspace path (e.g., "C:\Users\...\untitled3")
	BinPath           string // e.g., "npm" or "npx"
	BaseArgs          []string
	Timeout           time.Duration
	Env               []string
	DiscoveryProgress runners.DiscoveryProgress
}

func (j *JStester) SetDiscoveryProgress(progress runners.DiscoveryProgress) {
	j.DiscoveryProgress = progress
}

func (j *JStester) ListTests(projectPath string) ([]string, error) {
	if j.Timeout <= 0 {
		return nil, fmt.Errorf("Time is too short, please enter something larger than 0")
	}
	ctx, cancel := context.WithTimeout(context.Background(), j.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "npx", "jest", "--listTests")
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
	cmd.Dir = projectPath // Uses the passed parameter during discovery

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("JS test discovery failed: could not read command output: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("JS test discovery failed: %v | output: %s", err, stderr.String())
	}

	var tests []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line != "" {
			tests = append(tests, line)
			if j.DiscoveryProgress != nil {
				j.DiscoveryProgress(len(tests), -1, line)
			}
		}
	}
	scanErr := scanner.Err()
	waitErr := cmd.Wait()
	output := stderr.String()
	if scanErr != nil {
		return nil, fmt.Errorf("JS test discovery failed: could not read command output: %w | output: %s", scanErr, output)
	}
	if waitErr != nil {
		return nil, fmt.Errorf("JS test discovery failed: %v | output: %s", waitErr, output)
	}

	return tests, nil
}
