/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/Justi/projectseapig/factory"
	"github.com/spf13/cobra"
)

// detectCmd represents the detect command
var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "detects language you are using",
	Long: `SeaPig detects current project language you are using and returns the key word 
	needed to execute run and pig, if the language is not supported will throw an error`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("searching files.....")
		lang := factory.Lang()
		if len(lang) == 0 {
			fmt.Println("error: lang not detected, likely not supported")
			return
		}
		fmt.Println(lang)

	},
}

func init() {
	rootCmd.AddCommand(detectCmd)
}
