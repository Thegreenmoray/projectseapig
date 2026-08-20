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

	// 1. Identify valid test source roots (Java and/or Kotlin)
	var searchPaths []string
	javaRoot := filepath.Join(projectPath, "src", "test", "java")
	kotlinRoot := filepath.Join(projectPath, "src", "test", "kotlin")

	if _, err := os.Stat(javaRoot); err == nil {
		searchPaths = append(searchPaths, javaRoot)
	}
	if _, err := os.Stat(kotlinRoot); err == nil {
		searchPaths = append(searchPaths, kotlinRoot)
	}

	// Fallback to scanning the whole project path if standard paths aren't found
	if len(searchPaths) == 0 {
		searchPaths = append(searchPaths, projectPath)
	}

	// 2. Walk through all identified search paths
	for _, searchPath := range searchPaths {
		err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("test discovery timed out after %v while scanning file tree", g.Timeout)
			default:
			}

			if !info.IsDir() {
				var ext string
				name := info.Name()

				// Match Java or Kotlin test naming conventions
				if strings.HasSuffix(name, "Test.java") || strings.HasSuffix(name, "Tests.java") {
					ext = ".java"
				} else if strings.HasSuffix(name, "Test.kt") || strings.HasSuffix(name, "Tests.kt") {
					ext = ".kt"
				}

				if ext != "" {
					relPath, err := filepath.Rel(searchPath, path)
					if err != nil {
						return err
					}

					// Strip the specific extension (.java or .kt) and convert separators to dots
					cleanPath := strings.TrimSuffix(relPath, ext)
					fqcn := strings.ReplaceAll(cleanPath, string(os.PathSeparator), ".")

					tests = append(tests, fqcn)
				}
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("test discovery timed out after %v while scanning file tree", g.Timeout)
	}

	return tests, nil
}
