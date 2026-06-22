/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/Justi/projectseapig/factory"
	"github.com/spf13/cobra"
)

var path string

// detectCmd represents the detect command
var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "detects language you are using",
	Long: `SeaPig detects current project language you are using and returns the key word 
	needed to execute run and pig, if the language is not supported will throw an error`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("searching files.....")
		lang := factory.Lang(path)
		if len(lang) == 0 {
			cmd.Println("error: lang not detected, likely not supported")
			return
		}
		cmd.Println(lang)

	},
}

func init() {
	rootCmd.AddCommand(detectCmd)
	detectCmd.Flags().StringVarP(&path, "path", "p", "", "Path to the project")
}
