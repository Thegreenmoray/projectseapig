package gorunner

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type Gotester struct {
	BinPath           string // e.g., "go"
	ProjectPath       string // e.g., "C:\Users\...\testfolder"
	BaseArgs          []string
	Timeout           time.Duration
	DiscoveryTimeout  time.Duration
	DiscoveryProgress runners.DiscoveryProgress
	Env               []string
}

func (g *Gotester) SetDiscoveryProgress(progress runners.DiscoveryProgress) {
	g.DiscoveryProgress = progress
}

func (g *Gotester) ListTests(projectPath string) ([]string, error) {
	if g.Timeout <= 0 {
		return nil, fmt.Errorf("Time is too short, please enter something larger than 0")
	}
	discoveryTimeout := g.DiscoveryTimeout
	if discoveryTimeout <= 0 {
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
	stdout, err := listPackages.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not read go list output: %w", err)
	}
	var stderr strings.Builder
	listPackages.Stderr = &stderr
	if err := listPackages.Start(); err != nil {
		return nil, fmt.Errorf("could not start go list ./...: %w", err)
	}
	var packagePaths []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		packagePath := strings.TrimSpace(scanner.Text())
		if packagePath == "" {
			continue
		}
		packagePaths = append(packagePaths, packagePath)
		if g.DiscoveryProgress != nil {
			g.DiscoveryProgress(len(packagePaths), -1, packagePath)
		}
	}
	scanErr := scanner.Err()
	listErr := listPackages.Wait()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("Go test discovery timed out after %v: %w, Maybe there's a database/AI agent being called?, or otherwise a process being stalled?", discoveryTimeout, ctx.Err())
	}
	if scanErr != nil {
		return nil, fmt.Errorf("failed reading go list ./... output: %w", scanErr)
	}
	if listErr != nil {
		output := strings.TrimSpace(strings.Join(packagePaths, "\n") + "\n" + stderr.String())
		if strings.Contains(output, "does not contain main module") || strings.Contains(output, "go.mod file not found") {
			return nil, fmt.Errorf("no Go tests found in %s; Go tests must be func Test... declarations in *_test.go files, so check that --lang matches this project", projectPath)
		}
		return nil, fmt.Errorf("go list ./... failed: %w | output: %s", listErr, output)
	}

	if g.DiscoveryProgress != nil {
		g.DiscoveryProgress(0, len(packagePaths), "")
	}

	var tests []string
	for index, packagePath := range packagePaths {
		if g.DiscoveryProgress != nil {
			g.DiscoveryProgress(index, len(packagePaths), packagePath)
		}
		listPackageTests := exec.CommandContext(ctx, bin, "test", "-list", ".*", packagePath)
		listPackageTests.Dir = projectPath
		output, err := listPackageTests.CombinedOutput()
		if ctx.Err() != nil {
			return nil, fmt.Errorf("Go test discovery timed out after %v: %w", discoveryTimeout, ctx.Err())
		}
		if err != nil {
			return nil, fmt.Errorf("go test -list failed for %s: %w | output: %s", packagePath, err, strings.TrimSpace(string(output)))
		}
		if g.DiscoveryProgress != nil {
			g.DiscoveryProgress(index+1, len(packagePaths), packagePath)
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
