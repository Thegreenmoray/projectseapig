package cmd

import (
	"testing"
)

func TestRootCmd_ExecuteHelp(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Errorf("Expected root command execution with --help to succeed, got: %v", err)
	}
}
