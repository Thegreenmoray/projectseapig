package gorunner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Gotester struct {
	BinPath     string // e.g., "go"
	ProjectPath string // e.g., "C:\Users\...\testfolder"
	BaseArgs    []string
	Timeout     time.Duration
	Env         []string
}

func (g *Gotester) ListTests(projectPath string) ([]string, error) {
	discoveryTimeout := g.Timeout
	if g.Timeout <= 0 {
		return nil, fmt.Errorf("Time is too short, please enter something larger than 0")
	}
	if discoveryTimeout < 2*time.Minute {
		discoveryTimeout = 2 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()

	bin := g.BinPath
	if bin == "" {
		bin = "go"
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
	listPackages := exec.CommandContext(ctx, bin, "list", "./...")
	listPackages.Dir = projectPath
	packageOutput, err := listPackages.CombinedOutput()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("Go test discovery timed out after %v: %w, Maybe there's a database/AI agent being called?, or otherwise a process being stalled?", discoveryTimeout, ctx.Err())
	}
	if err != nil {
		if strings.Contains(string(packageOutput), "does not contain main module") || strings.Contains(string(packageOutput), "go.mod file not found") {
			return nil, fmt.Errorf("no Go tests found in %s; Go tests must be func Test... declarations in *_test.go files, so check that --lang matches this project", projectPath)
		}
		return nil, fmt.Errorf("go list ./... failed: %w | output: %s", err, strings.TrimSpace(string(packageOutput)))
	}

	var tests []string
	for _, packagePath := range strings.Fields(string(packageOutput)) {
		listPackageTests := exec.CommandContext(ctx, bin, "test", "-list", ".*", packagePath)
		listPackageTests.Dir = projectPath
		output, err := listPackageTests.CombinedOutput()
		if ctx.Err() != nil {
			return nil, fmt.Errorf("Go test discovery timed out after %v: %w", discoveryTimeout, ctx.Err())
		}
		if err != nil {
			return nil, fmt.Errorf("go test -list failed for %s: %w | output: %s", packagePath, err, strings.TrimSpace(string(output)))
		}
		for _, line := range strings.Split(string(output), "\n") {
			line = strings.TrimSpace(line)
			if parts := strings.Fields(line); len(parts) > 0 && strings.HasPrefix(parts[0], "Test") {
				tests = append(tests, packagePath+"::"+parts[0])
			}
		}
	}

	if len(tests) == 0 {
		return nil, fmt.Errorf("no Go tests found in %s; Go tests must be func Test... declarations in *_test.go files, so check that --lang matches this project", projectPath)
	}

	return tests, nil
}
