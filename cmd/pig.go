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

	"container/heap"

	"github.com/Justi/projectseapig/daemons"
	"github.com/Justi/projectseapig/factory"
	"github.com/Justi/projectseapig/logs"
	"github.com/Justi/projectseapig/runners"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var n int
var l string
var deep bool

type Pair struct {
	Time     int32
	Testname string
}

// FloatHeap is a min-heap of float32.
type IntHeap []Pair

// 1. Len is part of sort.Interface.
func (h IntHeap) Len() int { return len(h) }

// 2. Less is part of sort.Interface. Determines min vs max heap.
func (h IntHeap) Less(i, j int) bool { return h[i].Time > h[j].Time }

// 3. Swap is part of sort.Interface.
func (h IntHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

// 4. Push adds an element to the underlying slice.
// Pointer receiver is required because it modifies the slice's length.
func (h *IntHeap) Push(x any) {
	*h = append(*h, x.(Pair))
}

// 5. Pop removes the last element from the underlying slice.
// Pointer receiver is required because it modifies the slice's length.
func (h *IntHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// pigCmd represents the pig command
var pigCmd = &cobra.Command{
	Use:   "pig",
	Short: "runs the unit tester multiple times",
	Long: `runs unit tester the number of times you ask in the specified lang
	. WARNING this process can be Long and CPU intensive, you will be given a chance
	 to back out if you are not ready`,
	Run: func(cmd *cobra.Command, args []string) {
		loopFlag, _ := cmd.Flags().GetInt("loop")
		n = loopFlag

		cancontinue, tester := verification()
		if !cancontinue {
			return
		}

		daemon, err := factory.Daemontype(l, ".")
		if err != nil {
			log.Error().Err(err).Msg("failed to initialize daemon")
			return
		}

		log.Info().Msg(factory.Yellow + "sending the herd! this may take a while....." + factory.Reset)

		if deep {
			n = 100
		}

		tests, err := tester.ListTests(".")
		if err != nil || len(tests) == 0 {
			log.Error().Err(err).Msg("failed to list tests or no tests found")
			return
		}

		if n == 10 && factory.Cfg.Workers > 0 {
			n = factory.Cfg.Workers
		}

		h := &IntHeap{}
		heap.Init(h)

		totalExpectedResults := len(tests) * n
		b, _ := logs.NewBoltRepo("TestTime")
		hashmap, _ := b.Extractpigtime()
		c := make(chan []runners.TestResult, totalExpectedResults)

		for _, testName := range tests {
			samplebbolttime := hashmap[testName]
			if samplebbolttime == 0 {
				samplebbolttime = hashmap[fmt.Sprintf("%s_0", testName)]
			}

			for i := 0; i < n; i++ {
				heap.Push(h, Pair{Time: int32(samplebbolttime), Testname: testName})
			}
		}

		cpucores := runtime.GOMAXPROCS(0) * 2

		// Bug 1 Fix: Build dynamic slices without empty trailing slots
		var sliceofslices [][]string
		currentChunk := make([]string, 0, cpucores)

		for h.Len() > 0 {
			pair := heap.Pop(h).(Pair)
			currentChunk = append(currentChunk, pair.Testname)

			if len(currentChunk) == cpucores {
				sliceofslices = append(sliceofslices, currentChunk)
				currentChunk = make([]string, 0, cpucores)
			}
		}
		if len(currentChunk) > 0 {
			sliceofslices = append(sliceofslices, currentChunk)
		}

		bar := progressbar.NewOptions(totalExpectedResults,
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionSetWidth(15),
			progressbar.OptionSetDescription("[seapig] Running tests..."),
			progressbar.OptionShowCount(),
			progressbar.OptionShowIts(),
			progressbar.OptionSetItsString("tests"),
			progressbar.OptionOnCompletion(func() {
				fmt.Println("\n[seapig] Finished running test queue!")
			}),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerHead:    "[green]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)

		var wg sync.WaitGroup

		if err := daemon.Start(); err != nil {
			log.Error().Err(err).Msg("failed to start test daemon")
			return
		}
		pool, _ := ants.NewPoolWithFunc(cpucores, func(payload interface{}) {
			args := payload.(taskArgs)
			defer args.wg.Done()

			results, err := daemon.RunTests(args.testNames)
			if err != nil {
				log.Error().Err(err).Msgf("Daemon execution failed for batch: %v", args.testNames)
				return
			}

			if len(results) > 0 {
				c <- results
			}
			_ = bar.Add(len(args.testNames))
		})

		for _, testnames := range sliceofslices {
			wg.Add(1)
			_ = pool.Invoke(taskArgs{
				testNames: testnames,
				tester:    daemon,
				ch:        c,
				wg:        &wg,
			})
		}

		// Bug 2 Fix: Synchronous teardown ensures all results land in channel c before reading
		wg.Wait()
		close(c)
		pool.Release()
		if err := daemon.Stop(); err != nil {
			log.Error().Err(err).Msg("failed to stop test daemon")
		}

		errorOutputs := make(map[string]string)
		testing := make(map[string][]runners.TestResult)

		for testBatch := range c {
			for _, result := range testBatch {
				testing[result.Testname] = append(testing[result.Testname], result)
			}
		}

		repo, _ := logs.NewBoltRepo("seapig.db")
		defer repo.Close()

		results1(errorOutputs, repo, testing)
	}}

//a little messy up here, may want to break this up

// go implictly casts a struct as an interface if an interface is requested
type taskArgs struct {
	testNames []string
	tester    daemons.TestExecutor
	ch        chan<- []runners.TestResult
	wg        *sync.WaitGroup
}

func results1(errorOutputs map[string]string, repo *logs.BoltRepo, testing map[string][]runners.TestResult) {
	for testName, runs := range testing {
		batchResult := runners.Pig{
			Testname:    testName,
			Run:         runs,
			Errorlog:    errorOutputs[testName], // 👈 Grab error specific to THIS test
			Dateandtime: time.Now().Format(time.RFC3339),
		}

		runners.Results(&batchResult)

		if err := repo.SavePig(testName, &batchResult); err != nil {
			log.Error().Err(err).Msgf(factory.Red+"failed to save logs for %s", testName)
		}
	}
	log.Info().Msg(factory.Green + "The Herd has finished." + factory.Reset)
}

// relativilty self explaintory
func verification() (bool, runners.TestRunner) {
	if n <= 0 {
		fmt.Println("Invaild numbers detected, using default values")
	}
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
		n = factory.Cfg.Workers
	}

	return true, pig
}

func init() {
	rootCmd.AddCommand(pigCmd)
	//25 in 1.0 release but 10 for testing
	pigCmd.Flags().IntVarP(&n, "loop", "c", 25, "How many times you want to test")
	pigCmd.Flags().StringVarP(&l, "lang", "a", "", "Language to run tests for (go, python, java, js, other)")
	pigCmd.MarkFlagRequired("lang")
	pigCmd.Flags().BoolVarP(&deep, "deep", "d", false, "Run deep flake detection (100 loops)")
}
