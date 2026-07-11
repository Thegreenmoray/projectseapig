package gorunner

import (
	"testing"
	"time"
)

func TestGoListTests(t *testing.T) {
	g := Gotester{BinPath: "go",
		BaseArgs: []string{"test"},
		Timeout:  5 * time.Second,
	}
	tests, err := g.ListTests(".")
	if err != nil {
		t.Fatal(err)
	}

	if len(tests) == 0 {
		t.Fatal("expected at least one Go test")
	}
}

func TestGoRunTest(t *testing.T) {
	g := Gotester{BinPath: "go",
		BaseArgs: []string{"test"},
		Timeout:  5 * time.Second,
	}
	result, _ := g.RunTest("TestAdd")

	if result.Testname != "TestAdd" {
		t.Fatalf("expected TestAdd, got %s", result.Testname)
	}
}
