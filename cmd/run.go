package cmd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Justi/projectseapig/factory"
	"github.com/Justi/projectseapig/runners"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var lang string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute SeaPig on unit tests once on selected language",
	Long: `SeaPig does one round of unit test checking given a valid language,
a less costly run, just to be certain that it isn't just tests failing and ensuring 
that SeaPig is configured correctly.`,
	Run: func(cmd *cobra.Command, args []string) {
		pig, err := factory.Testtype(lang) // Assuming factory function matches your previous setup
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Sending in a pig into the %s trench....\n", lang)

		jobs := make(chan string)
		results := make(chan runners.TestResult)

		var collectionWg sync.WaitGroup
		var workerWg sync.WaitGroup

		// 1. Start test collection
		collectionWg.Add(1)
		go testcollection(pig, jobs, &collectionWg)

		// 2. Start the worker to process jobs
		workerWg.Add(1)
		go worker(pig, jobs, results, &workerWg)

		// 3. Monitor collection: Close jobs channel when collection is done
		go func() {
			collectionWg.Wait()
			close(jobs)
		}()

		// 4. Monitor workers: Close results channel when workers finish processing jobs
		go func() {
			workerWg.Wait()
			close(results)
		}()

		anyFailed := false

		// 5. Drain the results channel safely
		for result := range results {
			log.Info().Msgf("--- Test Name: %s ---", result.Testname)
			log.Info().Msgf("Passed: %t", result.Passed)
			log.Info().Msgf("Output:\n%s", result.Stdout)
			log.Info().Msgf("Duration: %v", result.Timetaken)

			if !result.Passed {
				anyFailed = true
			}
		}

		// 6. Evaluate final status after ALL tests have run
		if anyFailed {
			log.Info().Msg("Overall Summary: FAIL")
			os.Exit(1)
		}

		log.Info().Msg("Overall Summary: PASS")
		os.Exit(0)
	},
}

func testcollection(pig runners.TestRunner, jobs chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	names, err := pig.ListTests(".")
	if err != nil {
		log.Error().Err(err).Msg("Failed to discover tests")
		return
	}
	for _, name := range names {
		jobs <- name
	}
}

func worker(pig runners.TestRunner, jobs <-chan string, results chan<- runners.TestResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for testName := range jobs {
		start := time.Now()
		result, err := pig.RunTest(testName)
		result.Timetaken = time.Since(start)
		if err != nil {
			log.Error().Err(err).Str("test", testName).Msg("Error executing test runner system")
			continue
		}

		results <- result
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&lang, "lang", "l", "", "Language to run tests for (go, python, java, js)")
	runCmd.MarkFlagRequired("lang")
}
