package cli

import (
	"bytes"
	"testing"
)

func TestRunCLI_Info_MissingAsset(t *testing.T) {
	err := RunCLI([]string{"info"})
	if err == nil {
		t.Fatal("expected error for missing asset")
	}
}

func TestRunCLI_Info_UnknownAsset(t *testing.T) {
	err := RunCLI([]string{"info", "-asset", "DOESNOTEXIST9"})
	if err == nil {
		t.Fatal("expected error for unknown asset")
	}
}

func TestRunCLI_Info_InvalidFlag(t *testing.T) {
	err := RunCLI([]string{"info", "-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestRunCLI_Info_Valid(t *testing.T) {
	chdirToRepoRoot(t)

	output := captureStdout(t, func() {
		err := RunCLI([]string{"info", "-asset", "PETR4"})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("PETR4: data available from")) {
		t.Errorf("output = %q, want it to describe PETR4's available date range", output)
	}
}
