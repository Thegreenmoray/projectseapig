/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
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

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Excute SeaPig on unit tests once on selected language",
	Long: `SeaPig does one round of unit test checking given a vaild language,
	a less costly run, just to be certain that it isnt just tests failing and ensuring 
	that SeaPig is configured correctly.`,
	Run: func(cmd *cobra.Command, args []string) {
		pig, err := factory.Pigtype(lang)

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Sending in a pig into the %s trench....", lang)
		c := make(chan runners.TestResult)
		d := make(chan string)
		wg.Add(1)
		go worker(pig, d, c, &wg)
		go func() {
			wg.Wait()
			close(c)
		}()
		for result := range c {
			log.Info().Str("Test Name: %s\n", result.Testname).Msg("")
			log.Info().Bool("Passed: %t\n", result.Passed).Msg("")
			log.Info().Str("Output: %s\n", result.Stdout).Msg("")
			if result.Passed {
				log.Info().Msg("Overall: PASS")
				os.Exit(0)
			} else {
				log.Info().Msg("Overall: FAIL")
				os.Exit(1)
			}
		}
	},
}

func worker(pig runners.TestRunner, jobs <-chan string, results chan<- runners.TestResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for tests := range jobs {
		start := time.Now()
		result, errr := pig.RunTest(tests)
		result.Timetaken = time.Since(start)
		if errr != nil {
			continue
		}
		log.Info().
			Bool("passed", result.Passed).
			Str("test", result.Testname).
			Dur("time", result.Timetaken).
			Msg("test completed")
		results <- result
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&lang, "lang", "l", "", "Language to run tests for (go, python, java, js)")
	runCmd.MarkFlagRequired("lang")

}
