/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

//"encoding/json"
import (
	"bufio"
	"fmt"
	"os"
	"runtime"
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
var testLock sync.Mutex

// pigCmd represents the pig command
var pigCmd = &cobra.Command{
	Use:   "pig",
	Short: "runs the unit tester multiple times",
	Long: `runs unit tester the number of times you ask in the specified lang
	. WARNING this process can be Long and CPU intensive, you will be given a chance
	 to back out if you are not ready`,
	Run: func(cmd *cobra.Command, args []string) {
		loopFlag, _ := cmd.Flags().GetInt("loop")
		n = loopFlag //self explaintory just how many times you want to loop.

		cancontinue, tester := verification()
		if !cancontinue {
			return
		} //if user does not say yes or y, stop immedatly
		log.Info().Msg(factory.Yellow + "sending the herd! this may take a while....." + factory.Reset)

		if deep {
			n = 100
		}
		//finds all tests if any.
		tests, err := tester.ListTests(".")
		if err != nil {
			log.Error().Err(err).Msg("failed to list tests")
			return
		}

		if len(tests) == 0 {
			log.Info().Msg("no tests found")
		}
		//will be editable in config file or some other means
		if n == 10 && factory.Cfg.Defaultworkersize > 0 {
			n = factory.Cfg.Defaultworkersize
		}
		totalExpectedResults := len(tests) * n
		c := make(chan runners.TestResult, totalExpectedResults)
		// Inside your Command Run block:
		var wg sync.WaitGroup

		// 2. Instantiate a fixed PoolWithFunc.
		// The worker function is defined ONCE here.
		//ants is a more efficent goroutine, old way would spawn too many routines
		//using up too many resources, based on available cpu cores, accounts for VMs or CI/CD pipelines
		pool, _ := ants.NewPoolWithFunc(runtime.GOMAXPROCS(0)*2, func(payload interface{}) {
			//this is functional equvient to a lambda expression
			//recive the the data from the method below
			args := payload.(taskArgs)

			//defer just waits until we finish everything, even if it panics. prevents deadlocks.
			defer args.wg.Done()
			// if we do not do this, or put it at the bottom (or just lower) then the system can deadlock.
			start := time.Now()
			//allows process to build without risking early timeout, also prevents database deadlocks, file collisions, or shared port conflicts
			//by pausing for a moment each process.
			testLock.Lock()
			result, err := args.tester.RunTest(args.testName)
			testLock.Unlock()
			result.Timetaken = time.Since(start)
			//	log.Info().Msgf("%s", args.testName)
			if result.Testname == "" {
				result.Testname = args.testName // Guarantee it's never a blank string
			}
			//stores our error if we have one
			if err != nil {
				result.Stderr = err.Error()
			}
			//adds it to channel
			args.ch <- result

		})

		// 3. The dispatcher loop is now incredibly lightweight
		for _, testName := range tests {
			for i := 0; i < n; i++ {
				wg.Add(1)

				// Pass only the data payload. No new function allocation on the heap!
				_ = pool.Invoke(taskArgs{
					testName: testName,
					tester:   tester,
					ch:       c,
					wg:       &wg,
				})
			}
		}

		// 4. Teardown remains beautifully non-blocking
		go func() {
			wg.Wait()
			close(c)
			pool.Release()
		}()

		testing := make(map[string][]runners.TestResult)
		for f := range c {
			testing[f.Testname] = append(testing[f.Testname], f)
		}
		repo, _ := logs.NewBoltRepo("seapig.db")
		defer repo.Close()

		results1(repo, &testing)

	},
}

// go implictly casts a struct as an interface if an interface is requested
type taskArgs struct {
	testName string
	tester   runners.TestRunner
	ch       chan<- runners.TestResult
	wg       *sync.WaitGroup
}

//a little messy up here, may want to break this up

func results1(repo *logs.BoltRepo, testing *map[string][]runners.TestResult) {
	/*if len(*testing) == 0 {
		fmt.Println("DEBUG: The testing map is COMPLETELY EMPTY inside results1!")
		return
	}*/

	for testName, runs := range *testing {
		/*fmt.Printf("DEBUG: Found test in map: %s with %d runs\n", testName, len(runs))*/

		batchResult := runners.Pig{
			Testname: testName,
			Run:      runs,
		}

		runners.Results(&batchResult)

		// Snapshot exactly what is going into the DB
		//debugBytes, _ := json.MarshalIndent(batchResult, "", "  ")
		//fmt.Printf("DEBUG: Saving to DB for key [%s]:\n%s\n", testName, string(debugBytes))

		err := repo.SavePig(testName, batchResult)
		if err != nil {
			log.Error().Err(err).Msgf("failed to save logs for %s", testName)
		}
	}
}

// relativilty self explaintory
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
	//searches top of file for testpoint
	pig, err := factory.Testtype(l, ".")
	if err != nil {
		fmt.Println(err)
		return false, nil
	}

	if n <= 0 {
		n = factory.Cfg.Defaultworkersize
	}

	return true, pig
}

func init() {
	rootCmd.AddCommand(pigCmd)
	//25 in 1.0 release but 10 for testing
	pigCmd.Flags().IntVarP(&n, "loop", "c", 25, "How many times you want to test")
	pigCmd.Flags().StringVarP(&l, "lang", "a", "", "Language to run tests for (go, python, java, js)")
	pigCmd.MarkFlagRequired("lang")
	pigCmd.Flags().BoolVarP(&deep, "deep", "d", false, "Run deep flake detection (100 loops)")
}
