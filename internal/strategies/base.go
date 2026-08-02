package strategies

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lucasbz/backtests/internal/domain"
)

// availableStrategies maps a -strategy flag value to a constructor for the
// concrete domain.Strategy it should run, given the run's configured
// params. It's a map of constructors (not shared instances) so each run
// gets its own strategy state instead of leaking state across runs.
// Strategies no longer need a starting balance at construction time -
// Backtest owns the running balance and quantity sizing (see
// domain.Strategy.Decide), not here.
//
// params' meaning depends on the strategy:
//   - "buy-and-hold" and "two-candle-breakout" take no params - they
//     simply ignore whatever's passed (an empty/nil map is fine, same as
//     before this package's constructors took params at all).
//   - "sma-crossover" and "ema-crossover" expect "shortPeriod" and
//     "longPeriod" (see newCrossoverStrategy).
//   - "rsi-threshold" expects "period", "oversold" and "overbought" (see
//     newRSIThreshold).
var availableStrategies = map[string]func(params map[string]float64) (domain.Strategy, error){
	"buy-and-hold": func(params map[string]float64) (domain.Strategy, error) {
		return &BuyAndHold{}, nil
	},
	"two-candle-breakout": func(params map[string]float64) (domain.Strategy, error) {
		return &TwoCandleBreakout{}, nil
	},
	"sma-crossover": func(params map[string]float64) (domain.Strategy, error) {
		return newCrossoverStrategy("SMA Crossover", "sma", params)
	},
	"ema-crossover": func(params map[string]float64) (domain.Strategy, error) {
		return newCrossoverStrategy("EMA Crossover", "ema", params)
	},
	"rsi-threshold": func(params map[string]float64) (domain.Strategy, error) {
		return newRSIThreshold(params)
	},
}

// StrategyParam describes one parameter a strategy's constructor expects,
// for a UI to render an appropriately-labeled input. It is a UI hint only,
// not the source of truth for validation - the constructor (see
// newCrossoverStrategy/newRSIThreshold) is still what actually enforces
// correctness, including cross-field rules a flat per-param Min/Max can't
// express (e.g. shortPeriod < longPeriod, oversold < overbought). Those
// still surface as a normal 400 from POST /api/backtest, shown the same way
// any other backtest error already is today.
type StrategyParam struct {
	// Key is the key expected in strategyParams, e.g. "shortPeriod".
	Key string `json:"key"`
	// Label is a human-readable label, e.g. "Short period".
	Label string `json:"label"`
	// Default is a sensible prefilled value.
	Default float64 `json:"default"`
	// Min is the inclusive lower bound, for the input's min attribute.
	Min float64 `json:"min"`
	// Max is the inclusive upper bound, omitted from JSON (not null) when
	// unbounded.
	Max *float64 `json:"max,omitempty"`
	// Step is the input step attribute, e.g. 1 for a period.
	Step float64 `json:"step"`
}

// StrategyInfo is one entry in GET /api/strategies' response: a strategy's
// name plus what params (if any) it expects. Params is [] (never null),
// for strategies that take none (e.g. buy-and-hold).
type StrategyInfo struct {
	Name   string          `json:"name"`
	Params []StrategyParam `json:"params"`
}

// ptr returns a pointer to f, for building StrategyParam.Max literals inline
// (a plain &100 isn't valid Go for a float64 literal).
func ptr(f float64) *float64 {
	return &f
}

// strategyParams is a hand-authored table, parallel to availableStrategies
// (same rationale as that map: strategy-specific meaning doesn't derive
// cleanly from reflection, and every other registry in this codebase -
// stoploss, indicators - is similarly hand-documented rather than
// introspected). Strategies with no entry here (buy-and-hold,
// two-candle-breakout) take zero params.
var strategyParams = map[string][]StrategyParam{
	"sma-crossover": {
		{Key: "shortPeriod", Label: "Short period", Default: 10, Min: 1, Step: 1},
		{Key: "longPeriod", Label: "Long period", Default: 30, Min: 2, Step: 1},
	},
	"ema-crossover": {
		{Key: "shortPeriod", Label: "Short period", Default: 12, Min: 1, Step: 1},
		{Key: "longPeriod", Label: "Long period", Default: 26, Min: 2, Step: 1},
	},
	"rsi-threshold": {
		{Key: "period", Label: "RSI period", Default: 14, Min: 2, Step: 1},
		{Key: "oversold", Label: "Oversold threshold", Default: 30, Min: 0, Max: ptr(100), Step: 1},
		{Key: "overbought", Label: "Overbought threshold", Default: 70, Min: 0, Max: ptr(100), Step: 1},
	},
}

// AvailableStrategyInfo returns every registered strategy's name plus its
// declared params (see strategyParams), sorted by name - the same set and
// ordering as AvailableStrategyNamesList, but describing each strategy's
// expected strategyParams keys for a UI to render inputs from, rather than
// just the bare name. A strategy with no entry in strategyParams gets an
// empty (never nil) Params slice.
func AvailableStrategyInfo() []StrategyInfo {
	names := AvailableStrategyNamesList()
	infos := make([]StrategyInfo, 0, len(names))
	for _, name := range names {
		// Copy strategyParams[name] rather than reusing its backing array
		// directly, so a caller mutating one call's result (e.g. a
		// StrategyParam.Key) can't corrupt subsequent calls' output or the
		// package-level table itself - same fresh-instance rationale as
		// AvailableStrategyNamesList.
		params := append([]StrategyParam{}, strategyParams[name]...)
		infos = append(infos, StrategyInfo{Name: name, Params: params})
	}
	return infos
}

// AvailableStrategyNamesList returns every registered strategy name, sorted.
// Unlike AvailableStrategyNames (a comma-joined string meant for CLI help
// text), this is meant for callers that want the raw list, e.g. to serialize
// as a JSON array.
func AvailableStrategyNamesList() []string {
	names := make([]string, 0, len(availableStrategies))
	for name := range availableStrategies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func AvailableStrategyNames() string {
	return strings.Join(AvailableStrategyNamesList(), ", ")
}

// LoadStrategy builds a fresh instance of the named strategy, configured
// from params (see availableStrategies for what each strategy expects).
// Returns an error both for an unknown strategy name and for a
// strategy-specific invalid param (e.g. a missing period).
func LoadStrategy(strategyName string, params map[string]float64) (domain.Strategy, error) {
	newStrategy, ok := availableStrategies[strategyName]
	if !ok {
		return nil, fmt.Errorf("unknown strategy %q, available: %s", strategyName, AvailableStrategyNames())
	}
	return newStrategy(params)
}
