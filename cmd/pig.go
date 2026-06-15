/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Justi/projectseapig/factory"
	"github.com/spf13/cobra"
)

var n int
var l string
var deep bool

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

		input = strings.TrimSpace(strings.ToLower(input))
		if !(input == "y" || input == "yes") {
			fmt.Println("user cancelled process")
			return
		}

		fmt.Println("sending the herd! this may take a while.....")

		if deep {
			n = 100
		}

		for i := 0; i < n; i++ {
			factory.Pigtype(l)
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
