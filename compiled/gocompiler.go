package compiled

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/Justi/projectseapig/runners"
)

// Thankfully this wasnt too complex, so using llms wasnt too detrimental to the learning process.
// the worst was the commands, which I would have likely had to look up anyway, but everything else was very striaghtforward.
// the daemons on the other hand are something i will likely need to struggle through on my own, as they are more complex and require a deeper understanding of the language and the problem domain. I will likely need to spend more time on those, and possibly seek help from others or use llms to assist me in understanding the concepts and implementation details.
// at least just one of them anyway.
type GoCompiler struct {
	ProjectPath  string // package path consisting of tests
	CompiledPath string //compiled binary path
	Timeout      time.Duration
}

func (gc *GoCompiler) Compiletester() error {
	cmd := exec.Command("go", "test", "-c", "-o", gc.CompiledPath, gc.ProjectPath)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
func (gc *GoCompiler) Removetester() error {
	return os.Remove(gc.CompiledPath)
}

// dummy func to be edited later
func (gc *GoCompiler) RunTests(batchoftests []string) ([]runners.TestResult, error) {
	//ctx, cancel := context.WithTimeout(context.Background(), gc.Timeout)
	//defer cancel()
	//should add a timeout to prevent tests from running indefinitely, but for now this is fine.
	var results []runners.TestResult

	for _, testName := range batchoftests {
		start := time.Now()
		cmd := exec.Command(gc.CompiledPath, "-test.v", "-test.run", fmt.Sprintf("^%s$", testName))
		output, err := cmd.CombinedOutput()
		stop := time.Since(start)
		result := runners.TestResult{
			Testname:  testName,
			Passed:    err == nil,
			Stdout:    string(output),
			Timetaken: stop,
		}
		results = append(results, result)

	}
	return results, nil
}
