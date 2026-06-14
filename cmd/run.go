/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/Justi/projectseapig/factory"
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
		//excute pig
		_ = pig
		fmt.Printf("Sending in a pig into the %s trench....", lang)

	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&lang, "l", "lang", "la", "Language to run tests for (go, python, java, js)")
	runCmd.MarkFlagRequired("lang")

}
