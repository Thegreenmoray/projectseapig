/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Justi/projectseapig/factory"
	"github.com/Justi/projectseapig/logs"
	"github.com/Justi/projectseapig/runners"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var n int
var l string
var deep bool
var wg sync.WaitGroup

// pigCmd represents the pig command
var pigCmd = &cobra.Command{
	Use:   "pig",
	Short: "runs the unit tester multiple times",
	Long: `runs unit tester the number of times you ask in the specified lang
	. WARNING this process can be Long and CPU intensive, you will be given a chance
	 to back out if you are not ready`,
	Run: func(cmd *cobra.Command, args []string) {
		cancontinue, pig := verification()

		if !cancontinue {
			return
		}
		log.Info().Msg(factory.Yellow + "sending the herd! this may take a while....." + factory.Reset)

		if deep {
			n = 100
		}

		tests, err := pig.ListTests(".")
		if err != nil {
			log.Error().Err(err).Msg("failed to list tests")
			return
		} else if factory.Cfg.Defaultworkersize >= 0 {
			n = factory.Cfg.Defaultworkersize
		}
		totalExpectedResults := len(tests) * n
		c := make(chan runners.TestResult, totalExpectedResults)
		//ants is a more efficent goroutine, old way would spawn too many routines
		//using up too many resources
		pool, _ := ants.NewPool(n)
		var wg sync.WaitGroup

		for _, testName := range tests {

			for i := 0; i < n; i++ {
				wg.Add(1)
				// Capture testName safely for the goroutine closure
				tName := testName
				pool.Submit(func() {
					defer wg.Done()

					start := time.Now()
					result, err := pig.RunTest(tName)
					result.Timetaken = time.Since(start)
					if err == nil {
						c <- result
					}
				})
			}
		}
		go func() {
			wg.Wait()
			close(c)
			pool.Release()
		}()

		testing := make(map[string][]runners.TestResult)
		for f := range c {
			testing[f.Testname] = append(testing[f.Testname], f)
		}
		results(&testing)

	},
}

//a little messy up here, may want to break this up

func results(testing *map[string][]runners.TestResult) {

	for testName, runs := range *testing {
		batchResult := runners.Pig{
			Run: runs,
		}

		// Calculate pass/fail ratios and flakiness rate
		runners.Results(&batchResult)

		// Save this specific test's history directly into Bbolt!
		err := (&logs.BoltRepo{}).SavePig(testName, batchResult)
		if err != nil {
			log.Error().Err(err).Msgf("failed to save logs for %s", testName)
		}
	}

}

func verification() (bool, runners.TestRunner) {
	fmt.Println("warning! this very expensive to run, are you sure you want to do this (y/n)")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	if !(input == "y" || input == "yes") {
		fmt.Println("user cancelled process")
		return false, nil
	}
	if err != nil {
		fmt.Printf("failed to read input: %v\n", err)
		return false, nil
	}

	pig, err := factory.Pigtype(l)
	if err != nil {
		fmt.Println(err)
		return false, nil
	}

	if n <= 0 {
		fmt.Printf("%x is too small, try a bigger number", n)
		return false, nil
	}

	return true, pig
}

func init() {
	rootCmd.AddCommand(pigCmd)
	//25 in 1.0 release but 10 for testing
	pigCmd.Flags().IntVarP(&n, "loop", "c", 10, "How many times you want to test")
	pigCmd.Flags().StringVarP(&l, "lang", "a", "", "Language to run tests for (go, python, java, js)")
	pigCmd.MarkFlagRequired("lang")
	pigCmd.Flags().BoolVarP(&deep, "deep", "d", false, "Run deep flake detection (100 loops)")
}
