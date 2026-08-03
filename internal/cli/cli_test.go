package cli

import "testing"

func TestRunCLI_UnknownCommand(t *testing.T) {
	err := RunCLI([]string{"not-a-real-command"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}
