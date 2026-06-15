package gorunner

import "testing"

func TestListTests(t *testing.T) {
	g := Gotester{}
	tests, err := g.ListTests("./testdata")
	if err != nil {
		t.Fatal(err)
	}

	if len(tests) == 0 {
		t.Fatal("expected at least one test")
	}
}
