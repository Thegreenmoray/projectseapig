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

// pigCmd represents the pig command
var pigCmd = &cobra.Command{
	Use:   "pig",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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

		for i := 0; i < n; i++ {
			go factory.Pigtype(lang)
		}

	},
}

func init() {
	rootCmd.AddCommand(pigCmd)

	pigCmd.Flags().IntVarP(&n, "loop", "num", 10, "How many times you want to test")
	pigCmd.MarkFlagRequired("loop")
	runCmd.Flags().StringVarP(&l, "l", "lang", "la", "Language to run tests for (go, python, java, js)")
	runCmd.MarkFlagRequired("l")
}
