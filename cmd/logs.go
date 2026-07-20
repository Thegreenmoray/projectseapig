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
	Short: "View test logs",
	Long:  `View the logs of your test runs. This command reads from the local database and displays a summary of test results, including flakiness rates and pass/fail counts.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("logs called")
		db, err := bbolt.Open("seapig.db", 0600, &bbolt.Options{ReadOnly: true})
		if err != nil {
			log.Fatalf("Unable to open logs")
		}
		defer db.Close()

		if len(args) > 0 {
			// User specified a specific test, show detailed runs
			showDetailedLogs(db, args[0])
		} else {
			// User just typed 'seapig logs', show a summary of everything
			showSummary(db)
		}

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
			fmt.Printf("%-50s %-15.2f%% %-10s\n", p.Testname, p.Flakynessrate, status)
			return nil
		})
	})

	if err != nil {
		log.Fatalf("Error reading summary: %v", err)
	}
}

func showDetailedLogs(db *bbolt.DB, targetTest string) {
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("TestHistory"))
		if bucket == nil {
			fmt.Println("No logs found. Run some tests first!")
			return nil
		}
		//fairly self explaintory

		fmt.Printf("\n=== Detailed History for: %s ===\n\n", targetTest)
		found := false

		// We iterate through the bucket to find matches
		err := bucket.ForEach(func(k, v []byte) error {
			var p runners.Pig
			if err := json.Unmarshal(v, &p); err != nil {
				return err
			}

			// Check if this document belongs to the test the user asked for
			if p.Testname == targetTest {
				found = true

				// Print the batch overview header
				fmt.Printf("Batch Run [Flake Rate: %.2f%% | Pass: %d | Fail: %d]\n",
					p.Flakynessrate, p.PassCount, p.FailCount)
				fmt.Println("=====================================================================")

				// Loop through every individual run inside this batch (the marine snow!)
				for i, run := range p.Run {
					status := "PASSED"
					if !run.Passed {
						status = "FAILED"
					}

					fmt.Printf("  Run #%d | Status: %s | Duration: %v | Exit Code: %d\n",
						i+1, status, run.Timetaken, run.Exitcode)

					// If it failed and has stdout/stderr or metadata, expose it
					if !run.Passed {
						if run.Stdout != "" {
							fmt.Printf("    [Stdout]: %s\n", run.Stdout)
						}
						if run.Stderr != "" {
							fmt.Printf("    [Stderr]: %s\n", run.Stderr)
						}
						if len(run.Metadata) > 0 {
							fmt.Printf("    [Metadata]: %v\n", run.Metadata)
						}
					}
				}
				fmt.Println() // Add spacing between separate batch histories
			}
			return nil
		})

		if !found {
			fmt.Printf("No logged records found matching the test name: '%s'\n", targetTest)
			fmt.Println("Tip: Run 'projectseapig logs' without arguments to see available test names.")
		}

		return err
	})

	if err != nil {
		log.Fatalf("Error reading detailed logs: %v", err)
	}
}

func init() {
	rootCmd.AddCommand(logsCmd)
	//it just prints logs no need for anything else
}
