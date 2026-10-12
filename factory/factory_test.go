package factory

import (
	"fmt"
	"testing"
	"time"

	"github.com/Justi/projectseapig/daemons"
	"github.com/Justi/projectseapig/runners"
	"github.com/Justi/projectseapig/runners/gorunner"
	"github.com/Justi/projectseapig/runners/javarunner"
	"github.com/Justi/projectseapig/runners/jsrunner"
	"github.com/Justi/projectseapig/runners/pythonrunner"
)

func TestDaemon(t *testing.T) {
	java := "java"
	javarunner, err := Daemontype(java, ".")
	if err != nil {
		t.Fatal(err)
	}
	_, ok := javarunner.(daemons.TestExecutor)
	if ok {
		fmt.Printf("Is %s", java)
	}

	js := "js"
	jsrunner, errr := Daemontype(js, ".")
	if errr != nil {
		t.Fatal(errr)
	}
	_, okk := jsrunner.(daemons.TestExecutor)
	if okk {
		fmt.Printf("Is %s", js)
	}

	python := "python"
	pythonrunner, errr := Daemontype(python, ".")
	if errr != nil {
		t.Fatal(errr)
	}
	_, o := pythonrunner.(daemons.TestExecutor)
	if o {
		fmt.Printf("Is %s", python)
	}

	golang := "go"
	gorunner, errr := Daemontype(golang, ".")
	if errr != nil {
		t.Fatal(errr)
	}
	_, r := gorunner.(daemons.TestExecutor)
	if r {
		fmt.Printf("Is %s", golang)
	}

	ts := "ts"
	tsrunner, erre := Daemontype(ts, ".")
	if erre != nil {
		t.Fatal(erre)
	}
	_, td := tsrunner.(daemons.TestExecutor)
	if td {
		fmt.Printf("Is %s", golang)
	}

	kotlin := "kotlin"
	kotrunner, erreee := Daemontype(kotlin, ".")
	if erreee != nil {
		t.Fatal(erreee)
	}
	_, k := kotrunner.(daemons.TestExecutor)
	if k {
		fmt.Printf("Is %s", golang)
	}
	_, er := Daemontype("", ".")
	if er != nil {
		fmt.Printf("failed as expected")
	}
}

func TestFactory(t *testing.T) {
	java := "java"
	javarunner, err := Testtype(java, ".")
	if err != nil {
		t.Fatal(err)
	}
	_, ok := javarunner.(runners.TestRunner)
	if ok {
		fmt.Printf("Is %s", java)
	}

	js := "js"
	jsrunner, errr := Testtype(js, ".")
	if errr != nil {
		t.Fatal(errr)
	}
	_, okk := jsrunner.(runners.TestRunner)
	if okk {
		fmt.Printf("Is %s", js)
	}

	python := "python"
	pythonrunner, errr := Testtype(python, ".")
	if errr != nil {
		t.Fatal(errr)
	}
	_, o := pythonrunner.(runners.TestRunner)
	if o {
		fmt.Printf("Is %s", python)
	}

	golang := "go"
	gorunner, errr := Testtype(golang, ".")
	if errr != nil {
		t.Fatal(errr)
	}
	_, r := gorunner.(runners.TestRunner)
	if r {
		fmt.Printf("Is %s", golang)
	}

	_, er := Testtype("", ".")
	if er != nil {
		fmt.Printf("failed as expected")
	}

}

func TestKotlinAndTypeScriptDispatch(t *testing.T) {
	tests := []struct {
		language string
		check    func(daemons.TestExecutor) bool
		runner   func(runners.TestRunner) bool
	}{
		{
			language: "kotlin",
			check: func(executor daemons.TestExecutor) bool {
				daemon, ok := executor.(*daemons.JavaDaemon)
				return ok && daemon.IsKotlin && daemon.Socketpath != ""
			},
			runner: func(testRunner runners.TestRunner) bool {
				_, ok := testRunner.(*javarunner.Javatester)
				return ok
			},
		},
		{
			language: "ts",
			check: func(executor daemons.TestExecutor) bool {
				daemon, ok := executor.(*daemons.JsDaemon)
				return ok && daemon.IsTS && daemon.Socketpath != ""
			},
			runner: func(testRunner runners.TestRunner) bool {
				_, ok := testRunner.(*jsrunner.JStester)
				return ok
			},
		},
	}

	for _, test := range tests {
		t.Run(test.language, func(t *testing.T) {
			executor, err := Daemontype(test.language, ".")
			if err != nil {
				t.Fatal(err)
			}
			if !test.check(executor) {
				t.Fatalf("unexpected daemon for %s: %#v", test.language, executor)
			}

			testRunner, err := Testtype(test.language, ".")
			if err != nil {
				t.Fatal(err)
			}
			if !test.runner(testRunner) {
				t.Fatalf("unexpected test runner for %s: %#v", test.language, testRunner)
			}
		})
	}
}

func TestFactoryDispatchesAllSupportedLanguages(t *testing.T) {
	tests := []struct {
		language string
		check    func(daemons.TestExecutor) bool
		runner   func(runners.TestRunner) bool
	}{
		{
			language: "go",
			check: func(executor daemons.TestExecutor) bool {
				_, ok := executor.(*daemons.GoCompiler)
				return ok
			},
			runner: func(testRunner runners.TestRunner) bool {
				_, ok := testRunner.(*gorunner.Gotester)
				return ok
			},
		},
		{
			language: "java",
			check: func(executor daemons.TestExecutor) bool {
				daemon, ok := executor.(*daemons.JavaDaemon)
				return ok && !daemon.IsKotlin
			},
			runner: func(testRunner runners.TestRunner) bool {
				_, ok := testRunner.(*javarunner.Javatester)
				return ok
			},
		},
		{
			language: "kotlin",
			check: func(executor daemons.TestExecutor) bool {
				daemon, ok := executor.(*daemons.JavaDaemon)
				return ok && daemon.IsKotlin
			},
			runner: func(testRunner runners.TestRunner) bool {
				_, ok := testRunner.(*javarunner.Javatester)
				return ok
			},
		},
		{
			language: "js",
			check: func(executor daemons.TestExecutor) bool {
				daemon, ok := executor.(*daemons.JsDaemon)
				return ok && !daemon.IsTS
			},
			runner: func(testRunner runners.TestRunner) bool {
				_, ok := testRunner.(*jsrunner.JStester)
				return ok
			},
		},
		{
			language: "ts",
			check: func(executor daemons.TestExecutor) bool {
				daemon, ok := executor.(*daemons.JsDaemon)
				return ok && daemon.IsTS
			},
			runner: func(testRunner runners.TestRunner) bool {
				_, ok := testRunner.(*jsrunner.JStester)
				return ok
			},
		},
		{
			language: "python",
			check: func(executor daemons.TestExecutor) bool {
				_, ok := executor.(*daemons.PythonDaemon)
				return ok
			},
			runner: func(testRunner runners.TestRunner) bool {
				_, ok := testRunner.(*pythonrunner.Pythontester)
				return ok
			},
		},
	}

	for _, test := range tests {
		t.Run(test.language, func(t *testing.T) {
			executor, err := Daemontype(test.language, ".")
			if err != nil {
				t.Fatalf("Daemontype(%q) error = %v", test.language, err)
			}
			if !test.check(executor) {
				t.Errorf("Daemontype(%q) returned unexpected type %T", test.language, executor)
			}

			testRunner, err := Testtype(test.language, ".")
			if err != nil {
				t.Fatalf("Testtype(%q) error = %v", test.language, err)
			}
			if !test.runner(testRunner) {
				t.Errorf("Testtype(%q) returned unexpected type %T", test.language, testRunner)
			}
		})
	}
}

func TestDaemonsUseConfiguredTestTimeout(t *testing.T) {
	previousConfig := Cfg
	Cfg.Timeout = "23s"
	defer func() { Cfg = previousConfig }()

	for _, language := range []string{"go", "java", "kotlin", "python", "js", "ts"} {
		executor, err := Daemontype(language, ".")
		if err != nil {
			t.Fatalf("Daemontype(%q) error = %v", language, err)
		}
		var got time.Duration
		switch daemon := executor.(type) {
		case *daemons.GoCompiler:
			got = daemon.Timeout
		case *daemons.JavaDaemon:
			got = daemon.Timeout
		case *daemons.PythonDaemon:
			got = daemon.Timeout
		case *daemons.JsDaemon:
			got = daemon.Timeout
		default:
			t.Fatalf("Daemontype(%q) returned unexpected type %T", language, executor)
		}
		if got != 23*time.Second {
			t.Errorf("Daemontype(%q) timeout = %v, want %v", language, got, 23*time.Second)
		}
	}
}

func TestColors(t *testing.T) {
	tests := map[string]string{
		"Reset":  Reset,
		"Red":    Red,
		"Green":  Green,
		"Yellow": Yellow,
		"Blue":   Blue,
		"Bold":   Bold,
	}

	expected := map[string]string{
		"Reset":  "\033[0m",
		"Red":    "\033[31m",
		"Green":  "\033[32m",
		"Yellow": "\033[33m",
		"Blue":   "\033[34m",
		"Bold":   "\033[1m",
	}

	for name, val := range tests {
		if val != expected[name] {
			t.Errorf("%s: expected %q, got %q", name, expected[name], val)
		}
	}
}
