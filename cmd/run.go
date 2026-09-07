package cmd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Justi/projectseapig/daemons"
	"github.com/Justi/projectseapig/factory"
	"github.com/Justi/projectseapig/logs"
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
		pig, err := factory.Testtype(lang, ".") // Assuming factory function matches your previous setup
		//for now just assume its a daemon, we'll sort out go when ready
		daemon, errr := factory.Daemontype(lang, ".")
		if err != nil {
			fmt.Println(err)
			return
		}
		if errr != nil {
			fmt.Println(errr)
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
		go worker(daemon, jobs, results, &workerWg)

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
		repo, _ := logs.NewBoltRepo("seapig.db")
		// 5. Drain the results channel safely
		for result := range results {
			log.Info().Msgf("--- Test Name: %s ---", result.Testname)
			log.Info().Msgf("Passed: %t", result.Passed)
			log.Info().Msgf("Output:\n%s", result.Stdout)
			log.Info().Msgf("Duration: %v", result.Timetaken)

			if !result.Passed {
				anyFailed = true
				//if it fails we get a skewed runtime, this cannot be allowed.
				continue
			}
			batchResult := runners.Pig{
				Testname:    result.Testname,
				Dateandtime: time.Now().Format(time.RFC3339),
			}
			repo.SavePigtime(result.Testname, &batchResult)

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
	defer wg.Done() //input channel
	names, err := pig.ListTests(".")
	if err != nil {
		log.Error().Err(err).Msg("Failed to discover tests")
		return
	}
	for _, name := range names {
		jobs <- name
	}
}

// boot up the deamon/compiled lang were looking for here
func worker(pig daemons.Daemon, jobs <-chan string, results chan<- runners.TestResult, wg *sync.WaitGroup) {
	defer wg.Done() //output channel
	//no need to be explict about output
	names := []string{}
	for testName := range jobs {
		names = append(names, testName)
	}
	pig.StartDaemon()
	resultss, err := pig.RunTests(names)
	pig.StopDaemon()

	if err != nil {
		log.Error().Err(err).Msg("Error executing test runner system")
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
