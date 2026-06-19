package gorunner

import (
	"testing"
)

func TestGoDetect(t *testing.T) {
	g := Gotester{}
	idd, err := g.Detect(".")
	if err != nil {
		t.Fatal("expected Go project to be detected")
	}
	if idd < 10 {
		t.Fatal("expected Go project to be detected")
	}

}

func TestGoListTests(t *testing.T) {
	g := Gotester{}
	tests, err := g.ListTests(".")
	if err != nil {
		t.Fatal(err)
	}

	if len(tests) == 0 {
		t.Fatal("expected at least one Go test")
	}
}

func TestGoRunTest(t *testing.T) {
	g := Gotester{}
	result, _ := g.RunTest("TestAdd")

	if result.Testname != "TestAdd" {
		t.Fatalf("expected TestAdd, got %s", result.Testname)
	}
}
