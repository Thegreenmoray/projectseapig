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
