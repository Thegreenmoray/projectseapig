package factory

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Justi/projectseapig/daemons"
	"github.com/Justi/projectseapig/runners/gorunner"
	"github.com/Justi/projectseapig/runners/javarunner"
	"github.com/Justi/projectseapig/runners/jsrunner"
	"github.com/Justi/projectseapig/runners/pythonrunner"

	"github.com/Justi/projectseapig/runners"
)

// just go for now, will be updated later

func seapigServerPath(parts ...string) string {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join(append([]string{"servers"}, parts...)...)
	}
	repoRoot := filepath.Dir(filepath.Dir(sourceFile))
	return filepath.Join(append([]string{repoRoot, "servers"}, parts...)...)
}

func newDaemonSocketPath() (string, error) {
	file, err := os.CreateTemp("", "seapig-*.sock")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}

func Daemontype(lang string, projectPath string) (daemons.TestExecutor, error) {
	normalizedLang := strings.ToLower(lang)
	if normalizedLang == "go" {
		binName := "seapig_test_runner"
		if runtime.GOOS == "windows" {
			binName += ".exe"
		}
		return &daemons.GoCompiler{
			ProjectPath:  projectPath,
			CompiledPath: filepath.Join(projectPath, "bin", binName),
		}, nil
	}

	socketPath, err := newDaemonSocketPath()
	if err != nil {
		return nil, err
	}
	projectRoot, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}

	switch normalizedLang {
	case "java", "kotlin":
		return &daemons.JavaDaemon{
			TestPath:    filepath.Join(projectRoot, "src", "test"),
			ProjectRoot: projectRoot,
			DaemonPath:  seapigServerPath("build", "libs", "seapig-server-1.0-SNAPSHOT.jar"),
			IsKotlin:    normalizedLang == "kotlin",
			DaemonBase:  daemons.DaemonBase{Socketpath: socketPath},
		}, nil

	case "js", "ts":
		return &daemons.JsDaemon{
			NodePath:   projectRoot,
			DaemonPath: seapigServerPath("server.ts"),
			IsTS:       normalizedLang == "ts",
			DaemonBase: daemons.DaemonBase{Socketpath: socketPath},
		}, nil

	case "python":
		return &daemons.PythonDaemon{
			ProjectRoot: projectRoot,
			DaemonPath:  seapigServerPath("server.py"),
			DaemonBase:  daemons.DaemonBase{Socketpath: socketPath},
		}, nil

	default:
		return nil, errors.ErrUnsupported
	}
}

func Testtype(lang string, projectPath string) (runners.TestRunner, error) {
	timeout, err := time.ParseDuration(Cfg.Timeout)
	if err != nil || timeout <= 0 {
		timeout = 10 * time.Second
	}

	switch strings.ToLower(lang) {
	case "java", "kotlin":
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
	case "js", "ts":
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
