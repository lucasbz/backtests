# Technical indicators design plan

## Goal

Let strategies base entry/exit decisions on technical indicators (moving
averages to start: SMA and EMA crossovers, plus RSI as a third, differently-
shaped example) instead of only raw candle prices. Indicators must be
**reusable across multiple strategies** — the same `SMA`/`EMA`/`RSI`
implementation should back several different strategies — mirroring how
`domain.StopLoss` is a pluggable, registry-backed concept shared across the
codebase rather than something each strategy hand-rolls for itself.

Ship three strategies against this: `sma-crossover`, `ema-crossover`,
`rsi-threshold`, in parallel.

## Current architecture (for reference)

- `domain.Strategy` interface: `Name() string`, `Decide(candle Candle,
  position *Position, isLast bool) *Order`, called once per candle by
  `Backtest`'s traversal loop, in chronological order.
- Strategies that need history already keep it as **private struct state**,
  updated every `Decide` call — see `TwoCandleBreakout.window
  []domain.Candle`, trimmed to the last two candles via its `remember`
  helper. There's no interface-level concept of "history" or "indicators"
  today; each strategy that needs a rolling calculation reimplements it
  inline (currently only `TwoCandleBreakout`, with plain min/max over two
  candles - nothing as involved as a moving average yet).
- `strategies.LoadStrategy(name string) (domain.Strategy, error)` takes no
  parameters beyond the name - none of today's strategies are configurable
  per run. Contrast with `stoploss.LoadStrategy(name string, value float64)`,
  which already takes a single configurable number.
- `domain.StopLoss` is the closest existing precedent for a pluggable,
  registry-backed, reusable concept: `internal/stoploss/base.go`'s
  `availableStopLosses map[string]func(value float64) (domain.StopLoss,
  error)`, constructed by name + a single numeric `value`, exposed through
  `POST /api/backtest`'s `stopLoss: {type, value}` field.

## New architecture

### `internal/indicators` package

A small `Indicator` interface, fed one candle at a time (same shape as
`TwoCandleBreakout`'s `remember`, formalized and shared):

```go
package indicators

type Indicator interface {
    // Update feeds the next candle, in chronological order, into the
    // indicator's rolling state.
    Update(candle domain.Candle)
    // Value returns the indicator's current computed value and whether
    // enough candles have been seen yet to produce one - false during
    // warm-up (e.g. the first period-1 candles of an SMA).
    Value() (value float64, ready bool)
}
```

Concrete implementations: `SMA{Period int}`, `EMA{Period int}`, `RSI{Period
int}`, each with their own rolling-window/running-sum state, tested directly
against hand-computed reference sequences.

**Indicators compute in `float64`, not `money.Money`.** Unlike orders and
balances, indicator values are decision signals, not settled currency
amounts - `money.Money`'s integer-cents division is awkward for rolling
averages (already a pain point noted for `TotalFixedAmountStopLoss`'s
floor-division), and the tiny float precision loss here has no financial
consequence since indicators never directly become an `Order.Price`. Prices
going into an indicator (`candle.Close.AsMajorUnits()`, say) get converted
to `float64` at the boundary.

A registry, mirroring `internal/stoploss/base.go`'s shape but generalized to
multiple named params (a single `value float64` isn't enough for an SMA
crossover's two periods, or RSI's period + thresholds):

```go
var availableIndicators = map[string]func(params map[string]float64) (Indicator, error){
    "sma": func(params map[string]float64) (Indicator, error) { ... },
    "ema": func(params map[string]float64) (Indicator, error) { ... },
    "rsi": func(params map[string]float64) (Indicator, error) { ... },
}

func LoadIndicator(name string, params map[string]float64) (Indicator, error)
```

### Strategies compose indicators internally

Each new strategy constructs the `Indicator`(s) it needs from the registry
at load time, and updates them every `Decide` call - the reuse is at the
*implementation* level (`SMA`/`EMA`/`RSI` are written once, in
`internal/indicators`, and multiple strategies pull from the same registry),
not a generic "strategy built from arbitrary indicator config" DSL (out of
scope - see Open Questions).

`sma-crossover` and `ema-crossover` are structurally identical (two
indicators of the same kind, cross-over-triggered), so they share one
internal (unexported) implementation parameterized by which indicator
constructor backs each leg, registered twice under different names -
avoids duplicating the cross-detection logic:

```go
// internal/strategies/crossover.go (unexported, shared by sma-crossover
// and ema-crossover)
type crossoverStrategy struct {
    short, long   indicators.Indicator
    wasShortAbove *bool // nil until both indicators are first ready
    name          string
}

func (s *crossoverStrategy) Decide(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
    defer func() { s.short.Update(candle); s.long.Update(candle) }()

    shortVal, shortReady := s.short.Value()
    longVal, longReady := s.long.Value()
    if !shortReady || !longReady {
        return nil // still warming up
    }

    nowAbove := shortVal > longVal
    defer func() { s.wasShortAbove = &nowAbove }()
    if s.wasShortAbove == nil {
        return nil // first ready candle - no prior state to compare against, no cross yet
    }

    crossedUp := !*s.wasShortAbove && nowAbove
    crossedDown := *s.wasShortAbove && !nowAbove
    // ... buy on crossedUp while flat, sell on crossedDown while holding,
    // at candle.Close (or another documented price - TBD at implementation
    // time), same shape as TwoCandleBreakout's buy/sell branches.
}
```

`rsi-threshold` is a genuinely different shape (one indicator, two
configurable thresholds, not a crossover), so it's its own strategy type:
buys when RSI crosses below `oversold` (default e.g. 30), sells when RSI
crosses above `overbought` (default e.g. 70).

### Strategy parameters, end to end

Existing strategies (`buy-and-hold`, `two-candle-breakout`) take no
parameters; the new indicator-based ones need several (periods, thresholds).
Generalize `strategies.LoadStrategy` the same way `stoploss.LoadStopLoss`
already generalized beyond a bare name:

```go
func LoadStrategy(name string, params map[string]float64) (domain.Strategy, error)
```

Existing parameterless strategies' constructors simply ignore `params`
(and/or reject a non-empty one - TBD, see Open Questions). New ones read
their named params out of the map (e.g. `params["shortPeriod"]`), erroring
clearly on missing/invalid ones, same style as `stoploss`'s validation.

**API contract:** `BacktestRequest` gains an optional `strategyParams:
{[key: string]: number}`, alongside the existing `strategy: string` -
omitted/empty for parameterless strategies, required keys enforced by each
strategy's own constructor (not a generic schema - mirrors how `stopLoss`'s
`value` meaning is type-specific today). `openapi.yaml` documents each
strategy's expected keys in prose, same as it does for `stopLoss.value`'s
per-type meaning.

**CLI:** no per-key flags (wouldn't scale across differently-shaped param
sets) - strategy params are only settable via the existing `-config` JSON
file (a new `strategyParams` object alongside its current fields), which
already exists for exactly this kind of "more fields than are worth flags"
case.

**Frontend:** deferred to a second phase (see Implementation steps) -
`GET /api/strategies` currently returns a bare `string[]`; showing the right
input fields for a selected strategy needs it to describe what parameters
that strategy expects. Not designing that response shape now - flagged as
an open question to settle once the backend contract above is implemented
and proven out.

### Phase 1 has since shipped (update)

`internal/indicators`, the three new strategies, generalized `LoadStrategy`,
`strategyParams` on `POST /api/backtest`, and `-config`'s `strategyParams`
are all implemented and live - see git history. `GET /api/strategies` is
still a bare `string[]`, so the web UI has no way to enter `shortPeriod`/
`longPeriod`/etc for the three new strategies today; they're effectively
CLI/API-only. The rest of this doc designs the Phase 2 fix for that.

### `GET /api/strategies`'s new response shape

Replace the bare `string[]` with an array of small per-strategy
descriptors. Every current consumer is the frontend (`api/client.ts`'s
`getStrategies()`) - the CLI calls `strategies.AvailableStrategyNamesList()`
directly in-process, never through this endpoint - so this is a contained,
one-consumer breaking change to the endpoint, not a versioning concern.

```go
// internal/strategies/base.go

// StrategyParam describes one parameter a strategy's constructor expects,
// for a UI to render an appropriately-labeled input. It is a UI hint only,
// not the source of truth for validation - the constructor (see
// newCrossoverStrategy/newRSIThreshold) is still what actually enforces
// correctness, including cross-field rules a flat per-param Min/Max can't
// express (e.g. shortPeriod < longPeriod, oversold < overbought). Those
// still surface as a normal 400 from POST /api/backtest, shown the same
// way any other backtest error already is today.
type StrategyParam struct {
    Key     string  // key expected in strategyParams, e.g. "shortPeriod"
    Label   string  // human-readable label, e.g. "Short period"
    Default float64 // sensible prefilled value
    Min     float64 // inclusive lower bound, for the input's min attribute
    Max     *float64 // inclusive upper bound, nil if unbounded
    Step    float64  // input step attribute, e.g. 1 for a period
}

// StrategyInfo is one entry in GET /api/strategies' response: a strategy's
// name plus what params (if any) it expects. Params is [] (never null),
// for strategies that take none (e.g. buy-and-hold).
type StrategyInfo struct {
    Name   string          `json:"name"`
    Params []StrategyParam `json:"params"`
}
```

A hand-authored table, parallel to `availableStrategies` (same rationale as
that map: strategy-specific meaning doesn't derive cleanly from reflection,
and every other registry in this codebase - `stoploss`, `indicators` - is
similarly hand-documented rather than introspected):

```go
var strategyParams = map[string][]StrategyParam{
    // buy-and-hold, two-candle-breakout: omitted - zero params.
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

func AvailableStrategyInfo() []StrategyInfo { /* sorted by Name, Params: []StrategyParam{} when the map has no entry */ }
```

(`sma-crossover` defaults to 10/30 - a common short-range SMA crossover
pair that produces signals within a typical single-year backtest range in
this app; `ema-crossover` defaults to 12/26 - the classic MACD periods.
Both are just prefilled starting points, not enforced.)

`internal/api/api.go`'s `handleStrategies` changes from
`strategies.AvailableStrategyNamesList()` to
`strategies.AvailableStrategyInfo()`. `openapi.yaml`'s `GET /api/strategies`
response schema changes from a bare string array to an array of
`StrategyInfo` objects, with a worked example showing both a
zero-param and a multi-param strategy.

### Frontend: dynamic param inputs

- `api/client.ts`: `getStrategies(): Promise<string[]>` becomes
  `getStrategies(): Promise<StrategyInfo[]>`, with matching
  `StrategyInfo`/`StrategyParam` interfaces mirroring the Go JSON shape.
- `StrategyComparison.tsx`: keeps a `strategyParamValues: Record<string, string>`
  bit of local state (string, not number, same reasoning as the existing
  `stopLossValue` field - lets an input be legitimately empty rather than
  coerced to `0`). When the selected strategy's `params` is non-empty,
  render one labeled number input per param (reusing `fieldClasses`,
  `min`/`step` HTML attributes from the descriptor, prefilled from
  `Default` on strategy change) in the same row/grid area currently used
  for the stop-loss inputs' layout - stacked below it, not replacing it,
  since stop-loss still applies independently of which strategy is chosen.
  On submit, build `strategyParams` as `{[key]: Number(value)}` from
  whatever's currently in `strategyParamValues`, included only for
  strategies that have `params.length > 0` (omitted entirely otherwise,
  matching the API's existing "omit for parameterless strategies"
  contract).
- `utils/backtestForm.ts`: `canSubmitBacktest` gains a check that every
  param the selected strategy declares has a non-empty, valid numeric
  value in `strategyParamValues` before allowing submit - same shape as
  today's `isStopLossValid`, just iterating the strategy's declared params
  instead of a single fixed field. Cross-field rules (shortPeriod <
  longPeriod, etc.) are NOT re-validated client-side - left to the existing
  400-from-submit path, same as every other server-enforced rule in this
  form today (e.g. balance positivity).
- Changing the selected strategy resets `strategyParamValues` to that
  strategy's declared defaults (mirrors how changing `asset` already resets
  other form state in `AssetBrowser`'s effect).

## Implementation steps

**Phase 1 - backend + API contract:**

1. `internal/indicators` package: `Indicator` interface, `SMA`, `EMA`, `RSI`
   (+ direct unit tests against hand-computed reference sequences,
   including warm-up/`ready` behavior), and the `LoadIndicator` registry
   (+ tests).
2. Generalize `strategies.LoadStrategy`/`availableStrategies` to accept
   `params map[string]float64`; update `buy-and-hold`/`two-candle-breakout`
   registry entries and every call site (`cmd/cli.go`, `internal/api/api.go`,
   any test helpers) for the new signature - no behavior change for these
   two.
3. `internal/strategies/crossover.go` (shared, unexported) +
   `sma-crossover`/`ema-crossover` registry entries; `rsi-threshold` as its
   own type. Tests for all three via this package's existing
   `runStrategy`-style test helper pattern (see `two_candle_breakout_test.go`
   for the convention), including warm-up (no signal until indicators are
   ready) and at least one hand-traceable crossover/threshold-cross example
   per strategy.
4. `BacktestRequest.strategyParams`, wired through `internal/api/api.go`'s
   `handleBacktest` into `LoadStrategy`; `openapi.yaml` updated with the
   field and each new strategy's expected keys documented in prose.

**Phase 2 - CLI + frontend:** (steps 5 done; 6/7 are this update's scope)

5. ~~CLI: `strategyParams` object in the `-config` JSON schema~~ - done.
6. Backend: `StrategyParam`/`StrategyInfo` types + the hand-authored
   `strategyParams` table in `internal/strategies/base.go`,
   `AvailableStrategyInfo()`; `handleStrategies` switched over;
   `openapi.yaml`'s `GET /api/strategies` schema/example updated. Existing
   Go tests referencing the old `[]string` shape (`internal/api/api_test.go`,
   any `strategies` package tests) updated to match.
7. Frontend: `getStrategies()`'s return type + new `StrategyInfo`/
   `StrategyParam` interfaces in `api/client.ts`; dynamic per-param number
   inputs in `StrategyComparison.tsx` (reset to declared defaults on
   strategy change); `canSubmitBacktest` extended to require all declared
   params filled; `strategyParams` included in the challenger's
   `runBacktest` call whenever the selected strategy declares any.
   Existing tests/mocks referencing `getStrategies()`'s old shape updated;
   new tests for the dynamic-input rendering/reset/submit-gating behavior.

## Open questions / decisions made by default (flag if you disagree)

- Indicators compute in `float64`, not `money.Money` - a deliberate
  precision/ergonomics tradeoff, not an oversight (see "compute in float64"
  above).
- No Backtest-level pre-fetch of extra lead-in history before `-start`: an
  indicator simply warms up using whatever candles fall inside the
  requested range (e.g. a 50-day SMA produces no signal until the 50th
  candle of the run). Fetching extra history before the requested start
  date to avoid this warm-up gap is a legitimate future improvement, out of
  scope for v1.
- No generic "build a strategy from arbitrary indicator config" DSL/builder
  - only the three named, concretely-coded strategies above. A fully
  generic indicator-composition system is a much bigger feature and not
  what "reusable indicators" is asking for here; can revisit later if it
  turns out to be wanted.
- Crossover strategies exit/enter at `candle.Close` by default (simplest,
  matches how a crossover-detecting trader would realistically react to the
  candle that confirmed the cross) - open to revisiting per-strategy at
  implementation time if a different fill price reads better.
- `GET /api/strategies`'s response shape (see "Phase 2 update" above) is a
  hand-authored per-param descriptor table, UI-hint-only - it does not
  attempt to express cross-field validation rules (shortPeriod <
  longPeriod, oversold < overbought), which stay server-side-only, same as
  every other cross-field rule in this API today.
- This is a breaking change to `GET /api/strategies`'s response shape, but
  its only consumer is the frontend (the CLI never calls it), so no
  versioning/compat shim is needed.
