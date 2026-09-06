package factory

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Justi/projectseapig/compiled"
	"github.com/Justi/projectseapig/daemons"
	"github.com/Justi/projectseapig/runners/gorunner"
	"github.com/Justi/projectseapig/runners/javarunner"
	"github.com/Justi/projectseapig/runners/jsrunner"
	"github.com/Justi/projectseapig/runners/pythonrunner"

	"github.com/Justi/projectseapig/runners"
)

var Interpered HashSet[string] = *NewHashSet[string]()
var Compiled HashSet[string] = *NewHashSet[string]()

// just go for now, will be updated later
func Compilertype(lang string, projectPath string) *compiled.GoCompiler {
	binName := "seapig_test_runner"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	return &compiled.GoCompiler{
		ProjectPath:  projectPath,
		CompiledPath: filepath.Join(projectPath, "bin", binName),
	}
}

func Daemontype(lang string, projectPath string) daemons.Daemon {
	switch strings.ToLower(lang) {
	case "java":
		return &daemons.JavaDaemon{
			TestPath:   filepath.Join(projectPath, "src", "test", "java"),
			DaemonPath: filepath.Join("..", "projectseapig", "servers"),
			IsKotlin:   false,
		}
	case "kotlin":
		return &daemons.JavaDaemon{
			TestPath:   filepath.Join(projectPath, "src", "test", "kotlin"),
			DaemonPath: filepath.Join("..", "projectseapig", "servers"),
			IsKotlin:   true,
		}

	case "js":
		return &daemons.JsDaemon{
			NodePath:   projectPath,
			DaemonPath: filepath.Join("..", "projectseapig", "servers"),
			IsTS:       false,
		}
	case "ts":
		return &daemons.JsDaemon{
			NodePath:   projectPath,
			DaemonPath: filepath.Join("..", "projectseapig", "servers"),
			IsTS:       true,
		}

	case "python":
		return &daemons.PythonDaemon{
			ProjectRoot: projectPath,
			DaemonPath:  filepath.Join("..", "projectseapig", "servers"),
		}

	default:
		return nil
	}
}

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
