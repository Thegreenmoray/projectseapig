package factory

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Justi/projectseapig/runners/gorunner"
	"github.com/Justi/projectseapig/runners/javarunner"
	"github.com/Justi/projectseapig/runners/jsrunner"
	"github.com/Justi/projectseapig/runners/pythonrunner"

	"github.com/Justi/projectseapig/runners"
)

func Testtype(lang string, projectPath string) (runners.TestRunner, error) {
	timeout, err := time.ParseDuration(Cfg.Timeout)
	if err != nil || timeout <= 0 {
		timeout = 10 * time.Second
	}

	switch lang {
	case "java":
		// Set up smart defaults
		bin := "mvn"
		args := []string{"test"}

		// Check what kind of project layout we are dealing with
		if _, err := os.Stat(filepath.Join(projectPath, "build.gradle")); err == nil {
			bin = "gradle"
			// Check if the local wrapper script exists
			wrapper := "gradlew"
			if runtime.GOOS == "windows" {
				wrapper = "gradlew.bat"
			}
			if _, err := os.Stat(filepath.Join(projectPath, wrapper)); err == nil {
				bin = wrapper // Use the local wrapper if present
			}
		}

		return &javarunner.Javatester{
			BinPath:     bin,
			BaseArgs:    args,
			Timeout:     timeout,
			ProjectPath: projectPath, // Pass this down so RunTest knows where to execute
		}, nil
	case "js":
		return &jsrunner.JStester{
			BinPath:  "npm",
			BaseArgs: []string{"test", "--"},
			Timeout:  timeout,
		}, nil
	case "go":
		return &gorunner.Gotester{
			BinPath:  "go",
			BaseArgs: []string{"test"},
			Timeout:  timeout,
		}, nil
	case "python":
		// Using pytest as the default execution tool
		return &pythonrunner.Pythontester{
			BinPath:  "pytest",
			BaseArgs: []string{},
			Timeout:  timeout,
		}, nil
	default:
		return nil, errors.New("Lang not supported...")
	}
}
