# Strategy scan design plan

## Goal

Run a chosen strategy against every available asset (optionally restricted
to a year, same as `GET /api/assets`'s existing filter), compare each one
against the Buy & Hold baseline, and report which assets the strategy
actually beats the baseline on - "is this strategy winning anywhere, or
just on the couple of assets I happened to try by hand." Backend-first, per
your framing: this designs the batch-execution core, the API contract, and
a CLI command; a frontend leaderboard view is explicitly deferred (see
Implementation steps) - not designed here.

## Current architecture (for reference)

- `internal/backtest.Backtest{Asset, Start, End, Balance, Strategy,
  StopLoss}` / `.Run() (*Result, error)`: runs ONE strategy over ONE asset.
  `cmd/cli.go`'s `runCompare` already runs two `Backtest`s back-to-back
  (baseline + challenger) for a single asset and diffs their
  `ProfitPercentage` - a scan is the same comparison, just looped over
  every asset instead of one.
- `cotahist.ListAssets(year int) ([]string, error)`: the asset universe
  `GET /api/assets` and the CLI's asset list already pull from - `year ==
  0` means "no filter, everything ever imported." Reused as-is for the
  scan's asset universe (see Open Questions on why no other filtering
  mechanism is being added).
- **Correctness constraint that shapes this whole design**: `domain.
  Strategy` implementations carry mutable per-run state - `TwoCandleBreakout
  .window`, `crossoverStrategy`'s two `indicators.Indicator`s +
  `wasShortAbove`, `RSIThreshold`'s threshold-crossing bools,
  `EMATrendBreakout`'s window + EMAs. A single loaded `Strategy` instance
  can only ever safely back ONE `Backtest.Run()` call - reusing one across
  multiple assets (or handing the same instance to two goroutines
  concurrently) would leak one asset's trend/window state into the next
  asset's decisions, silently producing wrong results. This has never come
  up before because every existing caller (`runBacktest`, `runCompare`,
  `handleBacktest`) only ever runs one `Backtest` per loaded `Strategy`.
  A scan, looping over potentially hundreds of assets, is the first caller
  where this needs to be actively designed around rather than incidentally
  true - see "Fresh strategy per asset" below.
- By contrast, every `internal/stoploss` type (`PercentStopLoss`,
  `FixedAmountStopLoss`, `TotalFixedAmountStopLoss`, `NoStopLoss`) is
  stateless - `Check` only reads its own config fields (`Percent`,
  `Amount`) and the `position`/`candle` arguments, never mutates itself. A
  single loaded `StopLoss` instance is safe to share read-only across every
  asset and every concurrent goroutine in a scan.

## New architecture

### `internal/backtest/scan.go` (new file, same package as `Backtest`)

```go
// ScanParams configures a Scan run. StrategyName/StrategyParams (not a
// built domain.Strategy) is deliberate: Scan constructs a FRESH strategy
// instance per asset internally (see "Fresh strategy per asset" above) -
// taking a name+params instead of an instance makes it structurally
// impossible for a caller to accidentally share one mutable Strategy
// across assets, rather than relying on caller discipline to always
// remember to re-load it.
type ScanParams struct {
    Assets         []string
    Start, End     time.Time
    Balance        money.Money
    StrategyName   string
    StrategyParams map[string]float64
    // StopLoss is optional and, unlike Strategy, safe to share across
    // every asset/goroutine (see "stateless" note above) - built once by
    // the caller, same as Backtest.StopLoss today.
    StopLoss domain.StopLoss
}

// ScanResult is one asset's outcome: baseline (Buy & Hold) vs challenger
// (ScanParams.StrategyName), and whether the challenger won. Err is set
// (Baseline/Challenger/Won/Delta left zero) when running either backtest
// for this asset failed - a per-asset failure doesn't fail the whole scan,
// see "Per-asset errors" below.
type ScanResult struct {
    Asset      string
    Baseline   *Result
    Challenger *Result
    // Delta is Challenger.ProfitPercentage - Baseline.ProfitPercentage.
    Delta float64
    // Won is Delta > 0 (strictly - a tie is not a win).
    Won bool
    Err error
}

// scanConcurrency bounds how many assets are backtested at once. A fixed,
// modest constant rather than a runtime.NumCPU()-derived value - simple
// and deterministic-ish to reason about/test; revisit if scans over the
// full multi-hundred-asset universe prove too slow in practice.
const scanConcurrency = 8

// Scan runs ScanParams.StrategyName against every ScanParams.Assets entry,
// compared against a Buy & Hold baseline, with up to scanConcurrency
// assets running at once. Results are sorted by Delta descending (biggest
// winners first) with any errored entries last. The returned error is only
// for a caller-level misconfiguration (e.g. an invalid StrategyName/
// StrategyParams combination, checked once up front against a throwaway
// instance before spawning any work) - individual per-asset failures don't
// cause Scan itself to return an error, they show up as that asset's
// ScanResult.Err.
func Scan(params ScanParams) ([]ScanResult, error)
```

Sketch of `Scan`'s body (bounded worker pool via a semaphore channel +
`sync.WaitGroup` - no new dependency needed, `internal/backtest` doesn't
currently import `golang.org/x/sync` and this doesn't warrant adding it):

```go
func Scan(params ScanParams) ([]ScanResult, error) {
    // Fail fast on a bad strategy name/params before spawning any
    // goroutines - LoadStrategy is cheap (struct construction), so this
    // throwaway call is just validation, not reused for any asset's run.
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
            results[i] = scanOne(asset, params) // loads FRESH baseline+challenger Strategy instances internally
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
```

`scanOne(asset string, params ScanParams) ScanResult` loads a fresh
`buy-and-hold` `Strategy` and a fresh `params.StrategyName` `Strategy`
(via `strategies.LoadStrategy`, called twice, right here, per asset -
never hoisted outside this function), runs two `Backtest`s (sharing
`params.StopLoss` read-only, baseline gets no stop-loss - same convention
`StrategyComparison.tsx`'s frontend already follows: the baseline is
always a clean, stop-loss-free reference point), and returns the
`ScanResult` (with `.Err` set instead of `.Baseline`/`.Challenger`/
`.Delta`/`.Won` if either `Backtest.Run()` errored).

### `POST /api/scan`

A `POST`, not `GET` - same reasoning `POST /api/backtest` already
established: `strategyParams`/`stopLoss` are nested objects that don't fit
a query string cleanly.

```json
{
  "strategy": "sma-crossover",
  "strategyParams": { "shortPeriod": 10, "longPeriod": 30 },
  "stopLoss": { "type": "percent", "value": 5 },
  "start": "2015-01-01",
  "end": "2015-12-30",
  "balance": "10000.00",
  "year": 2015
}
```

`year` is optional (omit/`0` = every imported asset, exactly like
`GET /api/assets?year=`). Every other field mirrors `POST /api/backtest`'s
`BacktestRequest` exactly (same names/meanings) - this is deliberately
"the same request, minus `asset`, plus `year`," not a new shape to learn.

Response: an array, one entry per scanned asset, winners first (mirrors
`ScanResult`'s sort order). Deliberately lean - no per-operation
`operations` arrays (a several-hundred-asset response with full verbose
detail per asset would be a large payload for no benefit at this
aggregate, "where does this even work" level; drill into a specific
asset via the existing `POST /api/backtest`/CLI `compare` once a
candidate is found here):

```json
[
  {
    "asset": "PETR4",
    "baselineProfitPercentage": -27.77,
    "challengerProfitPercentage": 12.34,
    "challengerTotalOperations": 6,
    "delta": 40.11,
    "won": true
  },
  { "asset": "VALE3", "baselineProfitPercentage": 8.1, "challengerProfitPercentage": 8.1, "challengerTotalOperations": 0, "delta": 0, "won": false },
  { "asset": "XPTO3", "error": "loading candles for XPTO3: ..." }
]
```

`challengerTotalOperations` is included specifically so a "win" that's
really just "the strategy never traded and the baseline happened to lose
money" is visibly distinguishable from a win earned by actually trading -
worth surfacing at this summary level rather than only visible by drilling
into a specific asset's full result.

`internal/api/api.go`'s `handleScan`:
1. Parse/validate the body the same way `handleBacktest` does (required
   fields, `strategy`/`strategyParams` via a throwaway `LoadStrategy` call
   for early validation - same 400 semantics `handleBacktest` already has
   for an unknown strategy or a missing/invalid `strategyParams` key -
   `Scan` itself repeats this validation internally too, so this is purely
   about surfacing the error before doing any work, not a correctness
   requirement).
2. `year` optional int (`0` if absent, same as `GET /api/assets`'s
   handling).
3. `cotahist.ListAssets(year)` for the universe.
4. `stoploss.LoadStopLoss(...)` once if `stopLoss` present (shared
   read-only across the whole scan, per the "stateless" note above).
5. `backtest.Scan(backtest.ScanParams{...})`, map `[]ScanResult` to the
   lean JSON shape above (a small `scanResultJSON` shadow struct, same
   pattern `Result`'s own `resultJSON` already uses for money->float64
   conversion - though here it's ProfitPercentage/Delta, already plain
   floats, so the shadow struct is really just for choosing which fields
   are exposed + the `error` string).
6. `500` only for a `cotahist.ListAssets` failure or `Scan`'s own
   early-validation error; individual asset failures surface as `error`
   entries in the `200` response body, not as an overall failure status.

`openapi.yaml`: `POST /api/scan` documented, reusing `BacktestRequest`'s
existing field docs where the shape overlaps (via description cross-
references, same as this file already does elsewhere) plus the new `year`
field and a `ScanResult`/`ScanResponse` schema.

### CLI: `scan` command

Mirrors `compare`'s flags, minus `-asset`, plus `-year` (optional int, `0`
= all assets, matching `ListAssets`'s convention):

```
backtest scan -strategy sma-crossover -start 2015-01-01 -end 2015-12-30 -balance 10000.00 [-year 2015] [-stoploss-type percent -stoploss-value 5] [-config file.json]
```

Output, in the same concise `tabwriter`-table spirit as `compare`'s
operations table:

```
Scanning 187 assets: sma-crossover vs Buy & Hold (2015-01-01 to 2015-12-30)...

Asset    Baseline%   Challenger%   Delta     Won
PETR4    -27.77      12.34         +40.11    yes
VALE3       8.10        8.10        +0.00     no
...

Won on 42/184 assets (22.83%) - 3 assets failed to load, see below:
  XPTO3: loading candles for XPTO3: ...
```

(plain `yes`/`no` rather than a checkmark glyph, consistent with this
CLI's existing plain-ASCII table style elsewhere - see `printOperations`).
`cmd/main.go`'s `run` switch gains a `"scan"` case; usage/README updated
the same way `compare` was when it was added.

## Implementation steps

**Phase 1 - backend + API + CLI (this plan's full scope, per your
"backend-first" framing):**
1. `internal/backtest/scan.go`: `ScanParams`, `ScanResult`, `Scan`,
   `scanOne`, `scanConcurrency`.
2. `internal/backtest/scan_test.go`: a scan over a handful of assets
   produces the same per-asset numbers a sequential `Backtest.Run` would
   (proves concurrency doesn't leak state between assets - the core risk
   this design is built around); sort order (winners first, by Delta,
   errors last); an unknown `StrategyName`/invalid `StrategyParams` fails
   fast with no goroutines spawned; a deliberately-broken asset name
   produces a `ScanResult.Err` without failing the whole `Scan` call.
3. `internal/api/api.go`: `handleScan` + route; `internal/api/api_test.go`.
4. `openapi.yaml`: `POST /api/scan`.
5. `cmd/cli.go`: `runScan`; `cmd/main.go`'s switch; `cmd/cli_test.go`.
6. `README.md`: `scan` documented alongside `backtest`/`compare`.

**Phase 2 - frontend (explicitly deferred, not designed here):** a
leaderboard-style view consuming `POST /api/scan`'s response - not
designed now, flagged as future work once Phase 1 is proven out (mirrors
how `indicators.md` deferred its own frontend phase).

## Open questions / decisions made by default (flag if you disagree)

- "Won" = `Challenger.ProfitPercentage > Baseline.ProfitPercentage`,
  strictly - an exact tie is not a win.
- `scanConcurrency = 8`, a fixed constant, not derived from
  `runtime.NumCPU()` - simple and predictable; tune later if scans over
  the full asset universe prove too slow.
- Per-asset errors are captured on that `ScanResult` and don't fail the
  whole scan; the overall `Scan`/`POST /api/scan` call only errors on
  up-front misconfiguration (bad strategy/params) or an asset-listing
  failure.
- No `operations` detail in scan results - aggregate numbers only, by
  design (see "deliberately lean" above); use `compare`/
  `POST /api/backtest` to drill into a specific asset.
- Asset universe = `cotahist.ListAssets(year)`, same as `GET /api/assets` -
  no separate explicit-asset-list override for v1 (no evidence yet that
  the year filter alone is insufficient).
- The baseline never gets a stop-loss in a scan, same as `compare`'s
  existing behavior - only the challenger's `stopLoss` (if any) applies.
