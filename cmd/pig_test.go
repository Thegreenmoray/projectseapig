package cmd

import (
	"bytes"
	"os"
	"testing"
)

func TestPigCommand_VerificationFailures_Test(t *testing.T) {
	// --- Subtest 1: User chooses 'no' on the prompt ---
	t.Run("user cancels execution", func(t *testing.T) {
		// Simulate user typing "n" and pressing Enter
		inputReader, inputWriter, _ := os.Pipe()
		oldStdin := os.Stdin
		os.Stdin = inputReader
		defer func() { os.Stdin = oldStdin }()

		_, _ = inputWriter.Write([]byte("n\n"))
		_ = inputWriter.Close()

		outputBuf := &bytes.Buffer{}
		root := NewRootCmd()
		root.SetOut(outputBuf)

		// Pass valid language so it doesn't fail on Cobra flags, but triggers verification
		root.SetArgs([]string{"pig", "--lang", "go"})

		if err := root.Execute(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// --- Subtest 2: Invalid language selection ---
	t.Run("invalid language input", func(t *testing.T) {
		// Simulate user typing "y" to bypass the prompt
		inputReader, inputWriter, _ := os.Pipe()
		oldStdin := os.Stdin
		os.Stdin = inputReader
		defer func() { os.Stdin = oldStdin }()

		_, _ = inputWriter.Write([]byte("y\n"))
		_ = inputWriter.Close()

		outputBuf := &bytes.Buffer{}
		root := NewRootCmd()
		root.SetOut(outputBuf)

		// Pass an unsupported language string to force factory.Pigtype() to error out
		root.SetArgs([]string{"pig", "--lang", "invalid_lang"})

		if err := root.Execute(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
