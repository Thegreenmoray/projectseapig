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

	cmdf.SetArgs([]string{"detect", "--path", dir})
	buf := new(bytes.Buffer)
	cmdf.SetOut(buf)
	cmdf.SetErr(buf)

	if err := cmdf.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}

}
