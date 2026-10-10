package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Justi/projectseapig/daemons"
	"github.com/Justi/projectseapig/factory"
	"github.com/Justi/projectseapig/logs"
	"github.com/Justi/projectseapig/runners"
	"github.com/Justi/projectseapig/runners/gorunner"
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
that SeaPig is configured correctly.`,
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
			if goRunner, ok := pig.(*gorunner.Gotester); ok {
				goRunner.DiscoveryProgress = progress
			}
			return pig.ListTests(".")
		})
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
		resultCount := 0
		for result := range results {
			resultCount++
			_ = bar.Add(1)
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

		// 6. Evaluate final status after ALL tests have run
		if anyFailed || resultCount != len(names) {
			log.Info().Msg("Overall Summary: FAIL")
			if resultCount != len(names) {
				return fmt.Errorf("test runner returned %d results for %d discovered tests", resultCount, len(names))
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

func testSelectorForResult(resultName string, selectors []string) string {
	matched := ""
	for _, selector := range selectors {
		if (resultName == selector || strings.HasPrefix(resultName, selector+"::")) && len(selector) > len(matched) {
			matched = selector
		}
	}
	if matched != "" {
		return matched
	}
	return resultName
}

func listTestsWithProgress(listTests func(func(completed, total int, packagePath string)) ([]string, error)) ([]string, error) {
	_, _ = fmt.Fprint(os.Stdout, "\r[seapig] Discovering Go packages: 0 found")
	reportProgress := func(completed, total int, packagePath string) {
		if total < 0 {
			_, _ = fmt.Fprintf(os.Stdout, "\r[seapig] Discovering Go packages: %d found", completed)
		} else {
			renderPackageProgress(os.Stdout, completed, total)
		}
	}

	names, err := listTests(reportProgress)
	_, _ = fmt.Fprintln(os.Stdout)
	return names, err
}

func renderPackageProgress(writer io.Writer, completed, total int) {
	const width = 20
	if total < 0 {
		total = 0
	}
	if completed < 0 {
		completed = 0
	}
	if completed > total {
		completed = total
	}
	filled := 0
	if total > 0 {
		filled = completed * width / total
	}
	bar := "[" + strings.Repeat("=", filled) + strings.Repeat("-", width-filled) + "]"
	_, _ = fmt.Fprintf(writer, "\r[seapig] Scanning Go packages: %s %d/%d", bar, completed, total)
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
	runCmd.Flags().StringVarP(&lang, "lang", "l", "", "Language to run tests for (go, python, java, js)")
	runCmd.MarkFlagRequired("lang")
}
