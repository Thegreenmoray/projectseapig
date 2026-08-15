package daemons

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type JavaDaemon struct {
	daemonBase DaemonBase
	Socketpath string        // Path to the Unix socket for communication with the Java daemon
	Timeout    time.Duration //until a batch of tests are timed out
	DeamonPath string        //where the deamon is located
	TestPath   string        //where the test folder is located
	IsKotlin   bool          // Flag to indicate if the project is a Kotlin project
}

func (j *JavaDaemon) StartDaemon() error {

}

func (j *JavaDaemon) StopDaemon() error {

}

func (j *JavaDaemon) RunTests(testName []string) ([]runners.TestResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()

	bin := g.BinPath
	var args []string

	// 1. Build CLI args with performance flags (-q for quiet, -o for offline/no-remote-check)
	if strings.Contains(bin, "mvn") {
		args = append([]string{"test", "-q", "-o", "-B", "-Dtest=" + testName})
	} else {
		// Gradle execution
		args = append([]string{"test", "-q", "--tests", testName})
		if bin == "gradlew" {
			if runtime.GOOS == "windows" {
				bin = ".\\gradlew.bat"
			} else {
				bin = "./gradlew"
			}
		}
	}

	// 2. Resolve relative path for wrapper scripts
	absBin := bin
	if strings.HasPrefix(bin, ".") || bin == "gradlew.bat" {
		if resolved, err := filepath.Abs(filepath.Join(g.ProjectPath, bin)); err == nil {
			absBin = resolved
		}
	}

	// 3. Command setup
	cmd := exec.CommandContext(ctx, absBin, args...)
	cmd.Dir = g.ProjectPath

	if len(g.Env) > 0 {
		cmd.Env = g.Env
	}

	start := time.Now()
	out, err := cmd.CombinedOutput()
	d := []runners.TestResult{}
	return d, nil
}
