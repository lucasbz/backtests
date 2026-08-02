package backtest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lucasbz/backtests/internal/strategies"
)

// chdirToRepoRoot lets cotahist.LoadCandles find the real imported data
// under resources/cotahist. Mirrors the identically-named helper in
// internal/api/api_test.go and cmd/cli_test.go.
func chdirToRepoRoot(t *testing.T) {
	t.Helper()
	t.Chdir("../..")
}

func scanTestParams(assets []string, strategyName string, strategyParams map[string]float64) ScanParams {
	return ScanParams{
		Assets:         assets,
		Start:          time.Date(2015, 1, 2, 0, 0, 0, 0, time.UTC),
		End:            time.Date(2015, 12, 30, 0, 0, 0, 0, time.UTC),
		Balance:        newMoney(1000000), // R$10,000.00
		StrategyName:   strategyName,
		StrategyParams: strategyParams,
	}
}

// TestScan_MatchesSequentialBacktest is the core test this design is built
// around: with a stateful strategy (two-candle-breakout), a concurrent Scan
// must produce EXACTLY the same result per asset as running Backtest.Run()
// sequentially would - proving scanOne's fresh-strategy-per-asset instances
// don't leak state between assets.
func TestScan_MatchesSequentialBacktest(t *testing.T) {
	chdirToRepoRoot(t)

	assets := []string{"PETR4", "VALE3", "ITUB4"}
	params := scanTestParams(assets, "two-candle-breakout", nil)

	results, err := Scan(params)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != len(assets) {
		t.Fatalf("got %d results, want %d", len(results), len(assets))
	}

	byAsset := map[string]ScanResult{}
	for _, r := range results {
		byAsset[r.Asset] = r
	}

	for _, asset := range assets {
		scanResult, ok := byAsset[asset]
		if !ok {
			t.Fatalf("no ScanResult for %s", asset)
		}
		if scanResult.Err != nil {
			t.Fatalf("%s: ScanResult.Err = %v, want nil", asset, scanResult.Err)
		}

		baselineStrategy, err := strategies.LoadStrategy("buy-and-hold", nil)
		if err != nil {
			t.Fatalf("LoadStrategy(buy-and-hold): %v", err)
		}
		baselineBT := &Backtest{Asset: asset, Start: params.Start, End: params.End, Balance: params.Balance, Strategy: baselineStrategy}
		wantBaseline, err := baselineBT.Run()
		if err != nil {
			t.Fatalf("%s: sequential baseline Run: %v", asset, err)
		}

		challengerStrategy, err := strategies.LoadStrategy("two-candle-breakout", nil)
		if err != nil {
			t.Fatalf("LoadStrategy(two-candle-breakout): %v", err)
		}
		challengerBT := &Backtest{Asset: asset, Start: params.Start, End: params.End, Balance: params.Balance, Strategy: challengerStrategy}
		wantChallenger, err := challengerBT.Run()
		if err != nil {
			t.Fatalf("%s: sequential challenger Run: %v", asset, err)
		}

		assertResultsEqual(t, asset+" baseline", scanResult.Baseline, wantBaseline)
		assertResultsEqual(t, asset+" challenger", scanResult.Challenger, wantChallenger)

		wantDelta := wantChallenger.ProfitPercentage() - wantBaseline.ProfitPercentage()
		if !approxEqual(scanResult.Delta, wantDelta) {
			t.Errorf("%s: Delta = %v, want %v", asset, scanResult.Delta, wantDelta)
		}
		if scanResult.Won != (wantDelta > 0) {
			t.Errorf("%s: Won = %v, want %v", asset, scanResult.Won, wantDelta > 0)
		}
	}
}

func assertResultsEqual(t *testing.T, label string, got, want *BacktestResult) {
	t.Helper()
	if got == nil || want == nil {
		t.Fatalf("%s: got=%v want=%v, want both non-nil", label, got, want)
	}
	if got.TotalOperations != want.TotalOperations {
		t.Errorf("%s: TotalOperations = %d, want %d", label, got.TotalOperations, want.TotalOperations)
	}
	if got.Gains != want.Gains || got.Losses != want.Losses {
		t.Errorf("%s: Gains/Losses = %d/%d, want %d/%d", label, got.Gains, got.Losses, want.Gains, want.Losses)
	}
	if !approxEqual(got.ProfitPercentage(), want.ProfitPercentage()) {
		t.Errorf("%s: ProfitPercentage = %v, want %v", label, got.ProfitPercentage(), want.ProfitPercentage())
	}
	if got.EndingBalance.Amount() != want.EndingBalance.Amount() {
		t.Errorf("%s: EndingBalance = %d, want %d", label, got.EndingBalance.Amount(), want.EndingBalance.Amount())
	}
	if !approxEqual(got.MaxDrawdownPercentage(), want.MaxDrawdownPercentage()) {
		t.Errorf("%s: MaxDrawdownPercentage = %v, want %v", label, got.MaxDrawdownPercentage(), want.MaxDrawdownPercentage())
	}
}

// TestScan_SortOrder checks Scan's result ordering: every non-errored entry
// must come before every errored one, and among non-errored entries, Delta
// must be non-increasing (biggest winners first).
func TestScan_SortOrder(t *testing.T) {
	chdirToRepoRoot(t)

	// Out-of-order input, plus a bogus asset with no data (a guaranteed tie).
	assets := []string{"ITUB4", "VALE3", "PETR4", "ZZZZNODATA9"}
	params := scanTestParams(assets, "two-candle-breakout", nil)

	results, err := Scan(params)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != len(assets) {
		t.Fatalf("got %d results, want %d", len(results), len(assets))
	}

	seenErr := false
	prevDelta := 0.0
	for i, r := range results {
		if r.Err != nil {
			seenErr = true
			continue
		}
		if seenErr {
			t.Fatalf("result[%d] (%s) has no error but comes after an errored entry", i, r.Asset)
		}
		if i > 0 && results[i-1].Err == nil && r.Delta > prevDelta {
			t.Errorf("result[%d] (%s) Delta = %v, want <= previous entry's Delta %v (non-increasing order)", i, r.Asset, r.Delta, prevDelta)
		}
		prevDelta = r.Delta
	}
}

// TestScan_UnknownStrategyFailsFast checks Scan's up-front validation: an
// unknown StrategyName makes Scan return a non-nil error immediately, with
// an empty/nil results slice - proving no per-asset work was attempted.
func TestScan_UnknownStrategyFailsFast(t *testing.T) {
	params := scanTestParams([]string{"PETR4", "VALE3"}, "does-not-exist", nil)

	results, err := Scan(params)
	if err == nil {
		t.Fatal("expected an error for an unknown strategy name")
	}
	if len(results) != 0 {
		t.Errorf("results = %+v, want empty (no per-asset work attempted)", results)
	}
}

// TestScan_MissingRequiredStrategyParamFailsFast mirrors
// TestScan_UnknownStrategyFailsFast for a strategy that takes params
// (sma-crossover requires shortPeriod/longPeriod) but wasn't given any.
func TestScan_MissingRequiredStrategyParamFailsFast(t *testing.T) {
	params := scanTestParams([]string{"PETR4", "VALE3"}, "sma-crossover", nil)

	results, err := Scan(params)
	if err == nil {
		t.Fatal("expected an error for a missing required strategyParams key")
	}
	if len(results) != 0 {
		t.Errorf("results = %+v, want empty (no per-asset work attempted)", results)
	}
}

// writeSyntheticCandles writes a minimal, valid two-candle year file for
// ticker/year under dir (mirroring resources/cotahist/<TICKER>/
// <TICKER>_<YEAR>.json's shape).
func writeSyntheticCandles(t *testing.T, dir, ticker string, year int) {
	t.Helper()
	tickerDir := filepath.Join(dir, "resources", "cotahist", ticker)
	if err := os.MkdirAll(tickerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	data := `[
		{"date":"` + time.Date(year, 1, 2, 0, 0, 0, 0, time.UTC).Format("2006-01-02") + `","open":10,"high":10.5,"low":9.5,"avg":10,"close":10,"quantity":100,"volume":1000,"trades":10},
		{"date":"` + time.Date(year, 1, 5, 0, 0, 0, 0, time.UTC).Format("2006-01-02") + `","open":11,"high":11.5,"low":10.5,"avg":11,"close":11,"quantity":100,"volume":1000,"trades":10}
	]`
	path := filepath.Join(tickerDir, ticker+"_"+time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006")+".json")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// writeCorruptCandles writes an unparseable year file for ticker/year under
// dir, forcing Backtest.Run to fail for that ticker. A merely nonexistent
// ticker won't do - cotahist.LoadCandles treats missing files as "no data",
// not an error.
func writeCorruptCandles(t *testing.T, dir, ticker string, year int) {
	t.Helper()
	tickerDir := filepath.Join(dir, "resources", "cotahist", ticker)
	if err := os.MkdirAll(tickerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(tickerDir, ticker+"_"+time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006")+".json")
	if err := os.WriteFile(path, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// TestScan_PerAssetErrorDoesNotFailWholeScan proves a single asset's load
// failure (forced via a corrupt year file, see writeCorruptCandles) lands
// on that asset's ScanResult.Err, without making Scan itself return an
// error or affecting any other asset's result.
func TestScan_PerAssetErrorDoesNotFailWholeScan(t *testing.T) {
	dir := t.TempDir()
	writeSyntheticCandles(t, dir, "GOODA", 2015)
	writeSyntheticCandles(t, dir, "GOODB", 2015)
	writeCorruptCandles(t, dir, "BADTICKER", 2015)
	t.Chdir(dir)

	params := scanTestParams([]string{"GOODA", "BADTICKER", "GOODB"}, "buy-and-hold", nil)

	results, err := Scan(params)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}

	byAsset := map[string]ScanResult{}
	for _, r := range results {
		byAsset[r.Asset] = r
	}

	bad, ok := byAsset["BADTICKER"]
	if !ok {
		t.Fatal("no ScanResult for BADTICKER")
	}
	if bad.Err == nil {
		t.Error("BADTICKER: ScanResult.Err = nil, want a load error")
	}
	if bad.Baseline != nil || bad.Challenger != nil {
		t.Errorf("BADTICKER: Baseline/Challenger = %+v/%+v, want both nil", bad.Baseline, bad.Challenger)
	}

	for _, asset := range []string{"GOODA", "GOODB"} {
		good, ok := byAsset[asset]
		if !ok {
			t.Fatalf("no ScanResult for %s", asset)
		}
		if good.Err != nil {
			t.Errorf("%s: ScanResult.Err = %v, want nil (unaffected by BADTICKER's failure)", asset, good.Err)
		}
		if good.Baseline == nil || good.Challenger == nil {
			t.Errorf("%s: Baseline/Challenger = %+v/%+v, want both non-nil", asset, good.Baseline, good.Challenger)
		}
	}
}
