package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// writeRealAssetYearFixture copies a single real <TICKER>_<YEAR>.json file
// into destDir/resources/cotahist/<TICKER>/. Used instead of
// chdirToRepoRoot for scan tests: captureStdout only drains its pipe after
// the command finishes, and a scan over the full 2000+-ticker universe
// would produce enough output to deadlock that write before it gets there.
func writeRealAssetYearFixture(t *testing.T, destDir, ticker string, year int) {
	t.Helper()

	// The test binary's cwd is internal/cli/ (this package's directory), so
	// "../.." is the repo root - same relative step chdirToRepoRoot takes,
	// just resolved to an absolute path before any t.Chdir happens.
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}

	filename := fmt.Sprintf("%s_%d.json", ticker, year)
	srcPath := filepath.Join(repoRoot, "resources", "cotahist", ticker, filename)
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("reading real fixture %s: %v", srcPath, err)
	}

	dstDir := filepath.Join(destDir, "resources", "cotahist", ticker)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dstDir, filename), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestRunCLI_Scan_Valid(t *testing.T) {
	dir := t.TempDir()
	tickers := []string{"PETR4", "VALE3", "ITUB4"}
	for _, ticker := range tickers {
		writeRealAssetYearFixture(t, dir, ticker, 2015)
	}
	t.Chdir(dir)

	output := captureStdout(t, func() {
		err := RunCLI([]string{
			"scan",
			"-start", "2015-01-02",
			"-end", "2015-12-30",
			"-strategy", "two-candle-breakout",
			"-balance", "10000.00",
			"-year", "2015",
		})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	for _, want := range []string{
		fmt.Sprintf("Scanning %d assets", len(tickers)),
		"vs Buy & Hold", "Asset", "Baseline%", "Challenger%", "Delta", "Won",
		fmt.Sprintf("/%d assets (", len(tickers)),
	} {
		if !bytes.Contains([]byte(output), []byte(want)) {
			t.Errorf("output = %q, want it to contain %q", output, want)
		}
	}
	for _, ticker := range tickers {
		if !bytes.Contains([]byte(output), []byte(ticker)) {
			t.Errorf("output = %q, want it to contain %q", output, ticker)
		}
	}
}

// TestRunCLI_Scan_Verbose checks -v additionally prints a per-asset operations
// table for an asset the challenger actually traded on.
func TestRunCLI_Scan_Verbose(t *testing.T) {
	dir := t.TempDir()
	writeRealAssetYearFixture(t, dir, "PETR4", 2015)
	t.Chdir(dir)

	output := captureStdout(t, func() {
		err := RunCLI([]string{
			"scan",
			"-start", "2015-01-02",
			"-end", "2015-12-30",
			"-strategy", "two-candle-breakout",
			"-balance", "10000.00",
			"-year", "2015",
			"-v",
		})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	})

	if !bytes.Contains([]byte(output), []byte("=== PETR4: two-candle-breakout operations ===")) {
		t.Errorf("output = %q, want it to contain the PETR4 operations section header", output)
	}
	if !bytes.Contains([]byte(output), []byte("Buy date")) {
		t.Errorf("output = %q, want it to contain the operations table header", output)
	}
}

func TestRunCLI_Scan_MissingArgs(t *testing.T) {
	err := RunCLI([]string{"scan", "-start", "2015-01-02"})
	if err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestRunCLI_Scan_MissingBalance(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "two-candle-breakout",
	})
	if err == nil {
		t.Fatal("expected error for missing balance")
	}
}

func TestRunCLI_Scan_UnknownStrategy(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "does-not-exist",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

// TestRunCLI_Scan_BuyAndHoldChallengerRejected mirrors
// TestRunCLI_Compare_BuyAndHoldChallengerRejected: scan's baseline is always
// Buy & Hold, so it can't also be the challenger.
func TestRunCLI_Scan_BuyAndHoldChallengerRejected(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "buy-and-hold",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error when challenger strategy is buy-and-hold")
	}
}

func TestRunCLI_Scan_InvalidFlag(t *testing.T) {
	err := RunCLI([]string{"scan", "-not-a-real-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestRunCLI_Scan_InvalidDate(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "not-a-date",
		"-end", "2010-12-31",
		"-strategy", "two-candle-breakout",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for invalid start date")
	}
}

func TestRunCLI_Scan_InvalidEndDate(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "2010-01-01",
		"-end", "not-a-date",
		"-strategy", "two-candle-breakout",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for invalid end date")
	}
}

// TestRunCLI_Scan_EndBeforeStart mirrors TestRunCLI_Backtest_EndBeforeStart.
func TestRunCLI_Scan_EndBeforeStart(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "2010-12-31",
		"-end", "2010-01-01",
		"-strategy", "two-candle-breakout",
		"-balance", "10000.00",
	})
	if err == nil {
		t.Fatal("expected error for end before start")
	}
}

func TestRunCLI_Scan_InvalidBalance(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "two-candle-breakout",
		"-balance", "not-a-number",
	})
	if err == nil {
		t.Fatal("expected error for invalid balance")
	}
}

func TestRunCLI_Scan_ZeroBalance(t *testing.T) {
	err := RunCLI([]string{
		"scan",
		"-start", "2010-01-01",
		"-end", "2010-12-31",
		"-strategy", "two-candle-breakout",
		"-balance", "0",
	})
	if err == nil {
		t.Fatal("expected error for zero balance")
	}
}

// TestRunCLI_Scan_YearFiltersAssetUniverse: PETR4 has both a 2015 and 2016
// file, VALE3 only 2016, so -year 2015 must scan just PETR4 while -year
// 2016 scans both.
func TestRunCLI_Scan_YearFiltersAssetUniverse(t *testing.T) {
	dir := t.TempDir()
	writeRealAssetYearFixture(t, dir, "PETR4", 2015)
	writeRealAssetYearFixture(t, dir, "PETR4", 2016)
	writeRealAssetYearFixture(t, dir, "VALE3", 2016)
	t.Chdir(dir)

	scanFor := func(year string) string {
		return captureStdout(t, func() {
			err := RunCLI([]string{
				"scan",
				"-start", "2015-01-02",
				"-end", "2016-12-30",
				"-strategy", "two-candle-breakout",
				"-balance", "10000.00",
				"-year", year,
			})
			if err != nil {
				t.Fatalf("run: %v", err)
			}
		})
	}

	output2015 := scanFor("2015")
	if !bytes.Contains([]byte(output2015), []byte("Scanning 1 assets")) {
		t.Errorf("year=2015 output = %q, want it to contain %q", output2015, "Scanning 1 assets")
	}
	if !bytes.Contains([]byte(output2015), []byte("PETR4")) {
		t.Errorf("year=2015 output = %q, want it to contain PETR4", output2015)
	}
	if bytes.Contains([]byte(output2015), []byte("VALE3")) {
		t.Errorf("year=2015 output = %q, want it to NOT contain VALE3 (no 2015 data)", output2015)
	}

	output2016 := scanFor("2016")
	if !bytes.Contains([]byte(output2016), []byte("Scanning 2 assets")) {
		t.Errorf("year=2016 output = %q, want it to contain %q", output2016, "Scanning 2 assets")
	}
	for _, ticker := range []string{"PETR4", "VALE3"} {
		if !bytes.Contains([]byte(output2016), []byte(ticker)) {
			t.Errorf("year=2016 output = %q, want it to contain %q", output2016, ticker)
		}
	}
}
