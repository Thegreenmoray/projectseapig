package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Justi/projectseapig/daemons"
	"github.com/Justi/projectseapig/factory"
	"github.com/Justi/projectseapig/logs"
	"github.com/Justi/projectseapig/runners"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var lang string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute SeaPig on unit tests once on selected language",
	Long: `SeaPig does one round of unit test checking given a valid language,
a less costly run, just to be certain that it isn't just tests failing and ensuring 
that SeaPig is configured correctly.

Test discovery displays a live progress indicator, the number of tests found,
the current file or discovery phase, and elapsed time.`,
	Example: "projectseapig run --lang python",
	RunE: func(cmd *cobra.Command, args []string) error {
		pig, err := factory.Testtype(lang, ".")
		if err != nil {
			return err
		}
		daemon, err := factory.Daemontype(lang, ".")
		if err != nil {
			return err
		}

		fmt.Printf("Sending in a pig into the %s trench....\n", lang)

		names, err := listTestsWithProgress(func(progress func(completed, total int, packagePath string)) ([]string, error) {
			if reporter, ok := pig.(runners.DiscoveryProgressReporter); ok {
				reporter.SetDiscoveryProgress(progress)
			}
			return pig.ListTests(".")
		}, lang)
		if err != nil {
			log.Error().Err(err).Msg("Failed to discover tests")
			log.Info().Msg("Overall Summary: FAIL")
			return fmt.Errorf("failed to discover tests: %w", err)
		}
		if len(names) == 0 {
			log.Info().Msg("Overall Summary: FAIL")
			return errors.New("test discovery returned no tests")
		}

		repo, err := logs.NewBoltRepo("seapig.db")
		if err != nil {
			return fmt.Errorf("failed to open test history: %w", err)
		}
		defer repo.Close()

		jobs := make(chan string, len(names))
		results := make(chan runners.TestResult)

		var workerWg sync.WaitGroup

		// Start the worker to process discovered tests.
		workerWg.Add(1)
		go worker(daemon, jobs, results, &workerWg)

		bar := progressbar.NewOptions(len(names),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionSetWidth(15),
			progressbar.OptionSetDescription("[seapig] Running tests..."),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionSetItsString("tests"),
			progressbar.OptionOnCompletion(func() {
				fmt.Println("\n[seapig] Finished running test queue!")
			}),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerHead:    "[green]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)

		for _, name := range names {
			jobs <- name
		}
		close(jobs)
		go func() {
			workerWg.Wait()
			close(results)
		}()

		anyFailed := false
		completedTests := make(map[string]struct{}, len(names))
		for result := range results {
			_ = bar.Add(1)
			if selector, ok := matchingTestSelector(result.Testname, names); ok {
				completedTests[selector] = struct{}{}
			}
			//log.Info().Msgf("--- Test Name: %s ---", result.Testname)
			//log.Info().Msgf("Passed: %t", result.Passed)
			//log.Info().Msgf("Output:\n%s", result.Stdout)
			//log.Info().Msgf("Duration: %v", result.Timetaken)

			if !result.Passed {
				anyFailed = true
				if isTimedOutResult(result) {
					selector := testSelectorForResult(result.Testname, names)
					if err := repo.SaveTimedOutTest(selector); err != nil {
						log.Error().Err(err).Msgf("Failed to exclude timed-out test %s", selector)
					}
				}
				//if it fails we get a skewed runtime, this cannot be allowed.
				continue
			}
			batchResult := runners.Pig{
				Testname:    result.Testname,
				Run:         []runners.TestResult{result},
				Dateandtime: time.Now().Format(time.RFC3339),
			}
			if err := repo.SavePigtime(result.Testname, &batchResult); err != nil {
				log.Error().Err(err).Msgf("Failed to save test history for %s", result.Testname)
			}
		}

		// Some frameworks return one result per assertion or test method, not per discovery selector.
		missingTestCount := 0
		for _, name := range names {
			if _, completed := completedTests[name]; !completed {
				missingTestCount++
			}
		}
		// 6. Evaluate final status after ALL tests have run
		if anyFailed || missingTestCount > 0 {
			log.Info().Msg("Overall Summary: FAIL")
			if missingTestCount > 0 {
				return fmt.Errorf("test runner returned results for %d of %d discovered tests", len(names)-missingTestCount, len(names))
			}
			return errors.New("one or more tests failed")
		}

		log.Info().Msg("Overall Summary: PASS")
		return nil
	},
}

func isTimedOutResult(result runners.TestResult) bool {
	if result.TimedOut {
		return true
	}
	message := strings.ToLower(result.Stderr + "\n" + result.Stdout)
	return strings.Contains(message, "timed out") || strings.Contains(message, "timeout")
}

func matchingTestSelector(resultName string, selectors []string) (string, bool) {
	matched := ""
	for _, selector := range selectors {
		if (resultName == selector || strings.HasPrefix(resultName, selector+"::")) && len(selector) > len(matched) {
			matched = selector
		}
	}
	if matched != "" {
		return matched, true
	}
	return "", false
}

func testSelectorForResult(resultName string, selectors []string) string {
	if matched, ok := matchingTestSelector(resultName, selectors); ok {
		return matched
	}
	return resultName
}

func listTestsWithProgress(listTests func(func(completed, total int, packagePath string)) ([]string, error), language string) ([]string, error) {
	discoveryLabel := testDiscoveryLabel(language)
	startedAt := time.Now()
	progress := discoveryProgress{
		completed: 0,
		total:     -1,
		item:      discoveryWaitMessage(language),
	}
	var progressMu sync.Mutex
	stopRenderer := make(chan struct{})
	rendererStopped := make(chan struct{})

	go func() {
		defer close(rendererStopped)
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		frame := 0
		for {
			select {
			case <-ticker.C:
				progressMu.Lock()
				renderDiscoveryProgress(os.Stdout, progress, discoveryLabel, time.Since(startedAt), frame)
				progressMu.Unlock()
				frame++
			case <-stopRenderer:
				return
			}
		}
	}()

	reportProgress := func(completed, total int, packagePath string) {
		progressMu.Lock()
		progress.completed = completed
		progress.total = total
		if packagePath != "" {
			progress.item = packagePath
		}
		progressMu.Unlock()
	}

	names, err := listTests(reportProgress)
	close(stopRenderer)
	<-rendererStopped
	if err == nil {
		_, _ = fmt.Fprintf(os.Stdout, "\r\033[K[seapig] Discovery complete: %d %s test targets in %s\n", len(names), strings.ToLower(discoveryLabel), time.Since(startedAt).Round(time.Second))
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "\r\033[K[seapig] Discovery failed for %s after %s\n", discoveryLabel, time.Since(startedAt).Round(time.Second))
	}
	return names, err
}

type discoveryProgress struct {
	completed int
	total     int
	item      string
}

func testDiscoveryLabel(language string) string {
	switch strings.ToLower(language) {
	case "go":
		return "Go packages"
	case "java":
		return "Java tests"
	case "kotlin":
		return "Kotlin tests"
	case "js":
		return "JavaScript tests"
	case "ts":
		return "TypeScript tests"
	case "python":
		return "Python tests"
	default:
		return language + " tests"
	}
}

func discoveryWaitMessage(language string) string {
	switch strings.ToLower(language) {
	case "java", "kotlin":
		return "walking test source files"
	case "js", "ts":
		return "waiting for Jest to list test files"
	case "python":
		return "waiting for pytest collection"
	default:
		return "starting test discovery"
	}
}

func renderDiscoveryProgress(writer io.Writer, progress discoveryProgress, discoveryLabel string, elapsed time.Duration, frame int) {
	const width = 20
	item := progress.item
	if len([]rune(item)) > 60 {
		runes := []rune(item)
		item = "..." + string(runes[len(runes)-57:])
	}
	if progress.total < 0 {
		const indicatorWidth = 4
		position := frame % (width - indicatorWidth + 1)
		bar := strings.Repeat("-", position) + strings.Repeat("=", indicatorWidth) + strings.Repeat("-", width-position-indicatorWidth)
		_, _ = fmt.Fprintf(writer, "\r\033[K[seapig] Scanning %s: [%s] %d test targets found so far | %s | %s", discoveryLabel, bar, progress.completed, item, elapsed.Round(time.Second))
		return
	}
	if progress.completed < 0 {
		progress.completed = 0
	}
	if progress.completed > progress.total {
		progress.completed = progress.total
	}
	filled := 0
	if progress.total > 0 {
		filled = progress.completed * width / progress.total
	}
	bar := "[" + strings.Repeat("=", filled) + strings.Repeat("-", width-filled) + "]"
	item = filepath.Base(item)
	if item == "." {
		item = "preparing package scan"
	}
	_, _ = fmt.Fprintf(writer, "\r\033[K[seapig] Scanning %s: %s %d/%d | %s | %s", discoveryLabel, bar, progress.completed, progress.total, item, elapsed.Round(time.Second))
}

// boot up the deamon/compiled lang were looking for here
func worker(pig daemons.TestExecutor, jobs <-chan string, results chan<- runners.TestResult, wg *sync.WaitGroup) {
	defer wg.Done() //output channel
	//no need to be explict about output
	names := []string{}
	if err := pig.Start(); err != nil {
		results <- runners.TestResult{
			Testname: "<daemon>",
			Passed:   false,
			Stderr:   err.Error(),
		}
		return
	}
	defer func() {
		if err := pig.Stop(); err != nil {
			log.Error().Err(err).Msg("Failed to stop test daemon")
		}
	}()
	for testName := range jobs {
		names = append(names, testName)
	}

	resultss, err := pig.RunTests(names)

	if err != nil {
		log.Error().Err(err).Msg("Error executing test runner system")
		results <- runners.TestResult{
			Testname: "<daemon>",
			Passed:   false,
			Stderr:   err.Error(),
		}
		return
	}
	for _, result := range resultss {
		results <- result
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&lang, "lang", "l", "", "Language to run tests for (go, java, kotlin, js, ts, python)")
	runCmd.MarkFlagRequired("lang")
}
