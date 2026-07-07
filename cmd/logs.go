/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Justi/projectseapig/runners"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("logs called")
		db, err := bbolt.Open("seapig.db", 0600, &bbolt.Options{ReadOnly: true})
		if err != nil {
			log.Fatalf("Unable to open logs")
		}
		defer db.Close()

		showSummary(db)

	},
}

func showSummary(db *bbolt.DB) {
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("TestHistory"))
		if bucket == nil {
			fmt.Println("No logs found. Run some tests first!")
			return nil
		}

		fmt.Printf("%-50s %-15s %-10s\n", "TEST NAME", "FLAKE RATE", "PASS/FAIL")
		fmt.Println("---------------------------------------------------------------------------------")

		// Iterate through every record in the bucket
		return bucket.ForEach(func(k, v []byte) error {
			var p runners.Pig // Your Pig struct
			if err := json.Unmarshal(v, &p); err != nil {
				return err
			}

			// Format and print a clean summary line
			status := fmt.Sprintf("%d/%d", p.PassCount, p.FailCount+p.PassCount)
			fmt.Printf(" %-15.2f%% %-10s\n", p.Flakynessrate, status)
			return nil
		})
	})

	if err != nil {
		log.Fatalf("Error reading summary: %v", err)
	}
}

func init() {
	rootCmd.AddCommand(logsCmd)
	//it just prints logs no need for anything else
}
