package daemons

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Justi/projectseapig/runners"
)

// CommandRunner abstracts exec.Command for testing
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// RealCommandRunner is used in production
type RealCommandRunner struct{}

func (r *RealCommandRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// Thankfully this wasnt too complex, so using llms wasnt too detrimental to the learning process.
// the worst was the commands, which I would have likely had to look up anyway, but everything else was very striaghtforward.
// the daemons on the other hand are something i will likely need to struggle through on my own, as they are more complex and require a deeper understanding of the language and the problem domain. I will likely need to spend more time on those, and possibly seek help from others or use llms to assist me in understanding the concepts and implementation details.
// at least just one of them anyway.
type GoCompiler struct {
	ProjectPath  string // package path consisting of tests
	CompiledPath string //compiled binary path
	Timeout      time.Duration
	Runner       CommandRunner
	packageMu    sync.Mutex
	packageBins  map[string]string
	generated    []string
}

func (gc *GoCompiler) getRunner() CommandRunner {
	if gc.Runner == nil {
		return &RealCommandRunner{}
	}
	return gc.Runner
}

func (gc *GoCompiler) Start() error {
	if err := os.MkdirAll(filepath.Dir(gc.CompiledPath), 0755); err != nil {
		return fmt.Errorf("cannot create Go test binary directory: %w", err)
	}
	runner := gc.getRunner()
	output, err := runner.Run("go", "test", "-c", "-o", gc.CompiledPath, gc.ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to compile Go tests for %s: %w | output: %s", gc.ProjectPath, err, strings.TrimSpace(string(output)))
	}
	gc.packageMu.Lock()
	gc.packageBins = make(map[string]string)
	gc.generated = []string{gc.CompiledPath}
	gc.packageMu.Unlock()
	return nil
}

func (gc *GoCompiler) Stop() error {
	gc.packageMu.Lock()
	paths := append([]string(nil), gc.generated...)
	gc.packageBins = nil
	gc.generated = nil
	gc.packageMu.Unlock()

	var firstErr error
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (gc *GoCompiler) RunTests(batchoftests []string) ([]runners.TestResult, error) {
	if len(batchoftests) == 0 {
		return nil, nil
	}

	results := make([]runners.TestResult, 0, len(batchoftests))
	for _, selector := range batchoftests {
		packagePath, testName := splitGoTestSelector(selector)
		compiledPath := gc.CompiledPath
		if packagePath != "" {
			var err error
			compiledPath, err = gc.packageBinary(packagePath)
			if err != nil {
				return nil, err
			}
		}
		pattern := "^" + regexp.QuoteMeta(testName) + "$"
		start := time.Now()
		output, runErr := gc.getRunner().Run(compiledPath, "-test.v", "-test.run", pattern)
		passed, reported := goTestResult(output, testName)
		if runErr != nil {
			passed = false
		}

		result := runners.TestResult{
			Testname:  selector,
			Passed:    passed,
			Timetaken: time.Since(start),
			Stdout:    string(output),
		}
		if runErr != nil {
			result.Exitcode = 1
			result.Stderr = runErr.Error()
		} else if !reported {
			result.Stderr = fmt.Sprintf("compiled Go test binary did not report a result for %s", selector)
		}
		results = append(results, result)
	}

	return results, nil
}

func splitGoTestSelector(selector string) (string, string) {
	separator := strings.LastIndex(selector, "::")
	if separator < 0 {
		return "", selector
	}
	return selector[:separator], selector[separator+2:]
}

func (gc *GoCompiler) packageBinary(packagePath string) (string, error) {
	gc.packageMu.Lock()
	defer gc.packageMu.Unlock()
	if binary, ok := gc.packageBins[packagePath]; ok {
		return binary, nil
	}
	if gc.packageBins == nil {
		gc.packageBins = make(map[string]string)
	}

	hash := fnv.New32a()
	_, _ = hash.Write([]byte(packagePath))
	extension := filepath.Ext(gc.CompiledPath)
	base := strings.TrimSuffix(gc.CompiledPath, extension)
	binary := fmt.Sprintf("%s-%08x%s", base, hash.Sum32(), extension)
	if err := os.MkdirAll(filepath.Dir(binary), 0755); err != nil {
		return "", fmt.Errorf("cannot create Go package binary directory: %w", err)
	}
	output, err := gc.getRunner().Run("go", "test", "-c", "-o", binary, packagePath)
	if err != nil {
		return "", fmt.Errorf("failed to compile Go package %s: %w | output: %s", packagePath, err, strings.TrimSpace(string(output)))
	}
	gc.packageBins[packagePath] = binary
	gc.generated = append(gc.generated, binary)
	return binary, nil
}

func goTestResult(output []byte, testName string) (bool, bool) {
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 || fields[0] != "---" || fields[2] != testName {
			continue
		}
		switch strings.TrimSuffix(fields[1], ":") {
		case "PASS":
			return true, true
		case "FAIL":
			return false, true
		}
	}
	return false, false
}
