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

	"github.com/Justi/projectseapig/factory"
	"github.com/Justi/projectseapig/runners"
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
		fmt.Println("warning! this very expensive to run, are you sure you want to do this (y/n)")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("failed to read input: %v\n", err)
			return
		}

		if n <= 0 {
			fmt.Printf("%x is too small, try a bigger number", n)
			return
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if !(input == "y" || input == "yes") {
			fmt.Println("user cancelled process")
			return
		}

		fmt.Println("sending the herd! this may take a while.....")

		if deep {
			n = 100
		}

		c := make(chan runners.TestResult)

		for i := 0; i < n; i++ {
			pigg, err := factory.Pigtype(l)
			if err != nil {
				continue
			}
			wg.Add(1)
			go clump(pigg, c, &wg)

		}

		go func() {
			wg.Wait()
			close(c)
		}()
		for f := range c {
			fmt.Printf("Test Name: %s\n", f.Testname)
			fmt.Printf("Passed: %t\n", f.Passed)
			fmt.Printf("Output: %s\n", f.Stdout)

		}

	},
}

func init() {
	rootCmd.AddCommand(pigCmd)
	//25 in 1.0 release but 10 for testing
	pigCmd.Flags().IntVarP(&n, "loop", "c", 10, "How many times you want to test")
	pigCmd.Flags().StringVarP(&l, "lang", "a", "", "Language to run tests for (go, python, java, js)")
	pigCmd.MarkFlagRequired("lang")
	pigCmd.Flags().BoolVarP(&deep, "deep", "d", false, "Run deep flake detection (100 loops)")
}
