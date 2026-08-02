package backtest

import (
	"sort"
	"sync"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/strategies"
)

// baselineStrategyName is the strategy Scan always runs as each asset's
// baseline, mirroring cmd/cli.go's runCompare and
// frontend/src/components/StrategyComparison.tsx's BUY_AND_HOLD constant.
const baselineStrategyName = "buy-and-hold"

// ScanParams configures a Scan run. StrategyName/StrategyParams (not a
// built domain.Strategy) is deliberate: Scan constructs a FRESH strategy
// instance per asset internally (see scanOne) - taking a name+params
// instead of an instance makes it structurally impossible for a caller to
// accidentally share one mutable Strategy across assets, rather than
// relying on caller discipline to always remember to re-load it. See
// docs/plans/strategy-scan.md's "Current architecture" section for why this
// matters: domain.Strategy implementations carry mutable per-run state, and
// this scan runs many assets concurrently.
type ScanParams struct {
	Assets         []string
	Start, End     time.Time
	Balance        money.Money
	StrategyName   string
	StrategyParams map[string]float64
	// StopLoss is optional and, unlike Strategy, safe to share across every
	// asset/goroutine - every internal/stoploss implementation is stateless
	// (Check only reads its own config plus its arguments). Built once by
	// the caller, same as Backtest.StopLoss today. Only the challenger gets
	// it; the baseline never does, same convention runCompare/
	// StrategyComparison.tsx already follow.
	StopLoss domain.StopLoss
}

// ScanResult is one asset's outcome: baseline (Buy & Hold) vs challenger
// (ScanParams.StrategyName), and whether the challenger won. Err is set
// (Baseline/Challenger/Delta/Won left zero) when running either backtest
// for this asset failed - a per-asset failure doesn't fail the whole scan,
// see Scan's doc comment.
type ScanResult struct {
	Asset      string
	Baseline   *BacktestResult
	Challenger *BacktestResult
	// Delta is Challenger.ProfitPercentage() - Baseline.ProfitPercentage().
	Delta float64
	// Won is Delta > 0 (strictly - a tie is not a win).
	Won bool
	Err error
}

// scanConcurrency bounds how many assets are backtested at once. A fixed,
// modest constant rather than a runtime.NumCPU()-derived value - simple and
// deterministic-ish to reason about/test; revisit if scans over the full
// multi-hundred-asset universe prove too slow in practice.
const scanConcurrency = 8

// Scan runs ScanParams.StrategyName against every ScanParams.Assets entry,
// compared against a Buy & Hold baseline, with up to scanConcurrency assets
// running at once. Results are sorted with every non-errored entry first
// (by Delta descending, biggest winners first), then errored entries last.
// The returned error is only for a caller-level misconfiguration (e.g. an
// invalid StrategyName/StrategyParams combination, checked once up front
// against a throwaway instance before spawning any work) - individual
// per-asset failures don't cause Scan itself to return an error, they show
// up as that asset's ScanResult.Err.
func Scan(params ScanParams) ([]ScanResult, error) {
	// Fail fast on a bad strategy name/params before spawning any
	// goroutines - LoadStrategy is cheap (struct construction), so this
	// throwaway call is purely validation - its result is discarded, never
	// reused for any asset's actual run (see scanOne, which loads its own
	// fresh instance).
	if _, err := strategies.LoadStrategy(params.StrategyName, params.StrategyParams); err != nil {
		return nil, err
	}

	results := make([]ScanResult, len(params.Assets))
	sem := make(chan struct{}, scanConcurrency)
	var wg sync.WaitGroup
	for i, asset := range params.Assets {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, asset string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = scanOne(asset, params)
		}(i, asset)
	}
	wg.Wait()

	sort.SliceStable(results, func(i, j int) bool {
		if (results[i].Err == nil) != (results[j].Err == nil) {
			return results[i].Err == nil // non-errored entries first
		}
		return results[i].Delta > results[j].Delta
	})
	return results, nil
}

// scanOne runs the baseline (Buy & Hold) and challenger (params.StrategyName)
// backtests for a single asset, and compiles them into a ScanResult. It
// loads a FRESH domain.Strategy instance for each backtest - via
// strategies.LoadStrategy, called twice, right here - never hoisted outside
// this function or shared with any other asset's call: domain.Strategy
// implementations carry mutable per-run state, and scanOne runs
// concurrently across assets (see Scan), so reusing an instance across
// calls would silently leak one asset's state into another's decisions.
func scanOne(asset string, params ScanParams) ScanResult {
	baselineStrategy, err := strategies.LoadStrategy(baselineStrategyName, nil)
	if err != nil {
		return ScanResult{Asset: asset, Err: err}
	}
	challengerStrategy, err := strategies.LoadStrategy(params.StrategyName, params.StrategyParams)
	if err != nil {
		return ScanResult{Asset: asset, Err: err}
	}

	baselineBT := &Backtest{
		Asset:    asset,
		Start:    params.Start,
		End:      params.End,
		Balance:  params.Balance,
		Strategy: baselineStrategy,
		// No StopLoss: the baseline is always a clean, stop-loss-free
		// reference point, same as runCompare/StrategyComparison.tsx.
	}
	baselineResult, err := baselineBT.Run()
	if err != nil {
		return ScanResult{Asset: asset, Err: err}
	}

	challengerBT := &Backtest{
		Asset:    asset,
		Start:    params.Start,
		End:      params.End,
		Balance:  params.Balance,
		Strategy: challengerStrategy,
		StopLoss: params.StopLoss,
	}
	challengerResult, err := challengerBT.Run()
	if err != nil {
		return ScanResult{Asset: asset, Err: err}
	}

	delta := challengerResult.ProfitPercentage() - baselineResult.ProfitPercentage()
	return ScanResult{
		Asset:      asset,
		Baseline:   baselineResult,
		Challenger: challengerResult,
		Delta:      delta,
		Won:        delta > 0,
	}
}
