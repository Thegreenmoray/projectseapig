package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Justi/projectseapig/cmd"
)

func TestRootCommand(t *testing.T) {
	dir := t.TempDir()

	// Join connects the file to the proper os
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fake"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)

	cmdf := cmd.NewRootCmd()

	cmdf.SetArgs([]string{"list", dir})
	buf := new(bytes.Buffer)
	cmdf.SetOut(buf)
	cmdf.SetErr(buf)

	if err := cmdf.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}
}
//why isnt this being counted in coverage?

func TestPigcommand(t *testing.T) {
	dir := t.TempDir()
	modContent := []byte("module fake_project\n\ngo 1.21\n")
	err := os.WriteFile(filepath.Join(dir, "go.mod"), modContent, 0644)
	if err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// 3. Create a fake test file (must end in _test.go to be picked up by the toolchain)
	testFileContent := []byte(`package fake

import "testing"

func TestFakeExecution(t *testing.T) {
    // This test will always pass instantly
    if 1 + 1 != 2 {
        t.Errorf("Math broke.")
    }
}
`)
	err = os.WriteFile(filepath.Join(dir, "add_test.go"), testFileContent, 0644)
	if err != nil {
		t.Fatalf("failed to write fake test file: %v", err)
	}

	cmdf := cmd.NewRootCmd()

	cmdf.SetArgs([]string{"pig", "20", "--lang", "go", dir})

	if err := cmdf.Execute(); err != nil {
		t.Fatalf("pig command failed execution: %v", err)
	}

}
