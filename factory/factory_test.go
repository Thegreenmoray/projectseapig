package factory

import (
	"fmt"
	"testing"

	"github.com/Justi/projectseapig/runners"
)

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
