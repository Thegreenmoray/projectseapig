package javarunner

import (
	"context"
	"fmt"
	"os"

	"path/filepath"
	"strings"
	"time"
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
	if g.Timeout <= 0 {
		return nil, fmt.Errorf("Time is too short, please enter something larger than 0")
	}

	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	var tests []string
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

	// Target the standard Java test source directory
	testRoot := filepath.Join(projectPath, "src", "test", "java")
	searchPath := projectPath
	if _, err := os.Stat(testRoot); err == nil {
		searchPath = testRoot
	}

	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the timeout context was canceled during a long walk
		select {
		case <-ctx.Done():
			return fmt.Errorf("java test discovery timed out after %v while scanning file tree", g.Timeout)
		default:
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), "Test.java") {
			relPath, err := filepath.Rel(searchPath, path)
			if err != nil {
				return err
			}

			cleanPath := strings.TrimSuffix(relPath, ".java")
			fqcn := strings.ReplaceAll(cleanPath, string(os.PathSeparator), ".")

			tests = append(tests, fqcn)
		}
		return nil
	})

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("java test discovery timed out after %v while scanning file tree", g.Timeout)
	}

	return tests, err
}
