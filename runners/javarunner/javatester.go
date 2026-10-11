package javarunner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
)

var (
	sourcePackagePattern = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)`)
	kotlinClassPattern   = regexp.MustCompile(`\b(?:class|object)\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

type Javatester struct {
	BinPath           string   // e.g., "mvn" or "gradlew"
	BaseArgs          []string // e.g., []string{"test"}
	Timeout           time.Duration
	Env               []string
	ProjectPath       string // Added to ensure cmd.Dir points to the right spot
	DiscoveryProgress runners.DiscoveryProgress
}

func (g *Javatester) SetDiscoveryProgress(progress runners.DiscoveryProgress) {
	g.DiscoveryProgress = progress
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

			if g.DiscoveryProgress != nil {
				g.DiscoveryProgress(len(tests), -1, path)
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("test discovery timed out after %v while scanning file tree", g.Timeout)
			default:
			}

			if !info.IsDir() {
				name := strings.ToLower(info.Name())
				ext := filepath.Ext(name)
				if (ext != ".java" && ext != ".kt") ||
					(!strings.HasSuffix(name, "test"+ext) && !strings.HasSuffix(name, "tests"+ext)) {
					return nil
				}

				source, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("cannot read test source %q: %w", path, err)
				}

				relPath, err := filepath.Rel(searchPath, path)
				if err != nil {
					return err
				}
				className := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
				if ext == ".kt" {
					className = kotlinTestClassName(source, className)
				}

				packageName := sourcePackagePattern.FindSubmatch(source)
				if len(packageName) > 1 {
					tests = append(tests, string(packageName[1])+"."+className)
				} else {
					relativeDir := filepath.Dir(relPath)
					if relativeDir != "." {
						className = strings.ReplaceAll(relativeDir, string(os.PathSeparator), ".") + "." + className
					}
					tests = append(tests, className)
				}
				if g.DiscoveryProgress != nil {
					g.DiscoveryProgress(len(tests), -1, path)
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

func kotlinTestClassName(source []byte, fileName string) string {
	for _, match := range kotlinClassPattern.FindAllSubmatch(source, -1) {
		className := string(match[1])
		lowerClassName := strings.ToLower(className)
		if strings.HasSuffix(lowerClassName, "test") || strings.HasSuffix(lowerClassName, "tests") {
			return className
		}
	}

	return fileName + "Kt"
}
