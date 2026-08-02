# Code Review Findings & Remediation Plan

Source: full-tree review by the `code-reviewer` agent, 2026-08-01. Covers
backend (Go) and frontend (React/TypeScript) as they stand after the
stop-loss/OCO feature, the Ticker→Asset rename, and the Tailwind rewrite.

Findings are ordered most-severe first. Each has a proposed fix; none of
these have been implemented yet — this doc is the plan.

## 1. [Medium] Path traversal via unsanitized `asset` input

`internal/api/api.go` (`handleInfo` ~line 56, `handleBacktest` ~line 209)
passes the request's `asset`/`ticker` string straight through to
`internal/cotahist/cotahist.go`:

- `LoadCandlesFrom` (~line 33): `filepath.Join(dir, asset, fmt.Sprintf("%s_%d.json", asset, year))`
- `DateRangeFrom` (~line 75): `filepath.Join(dir, asset)` then `os.ReadDir(assetDir)`

No validation exists anywhere in the pipeline — `openapi.yaml`'s `asset`
schema is a bare `type: string`. Confirmed via a temporary test that values
containing `/` and `..` segments are accepted and used unmodified to build
the filesystem path. This gives a caller, at minimum, an existence/
permission-error oracle for arbitrary paths (via the differing error
strings `cotahist.go` returns — `"no data found"` vs `"reading %s: %w"` vs
a JSON parse error), and potential data disclosure if a matching
`<name>_<year>.json` exists along a traversed path.

**Fix:** validate `asset`/`ticker` against a strict allowlist
(e.g. `^[A-Z0-9]+$`) at the API boundary, before it reaches `cotahist`.

## 2. [Medium] `ParseMoney`/`MoneyFromFloat` accept `Inf`/`NaN` and silently saturate

`internal/domain/candle.go` lines 21-42. `strconv.ParseFloat` parses
`"Inf"`/`"+Inf"`/`"-Inf"`/`"NaN"` without error; `math.Round(f*100)` then
`int64(...)` on `+Inf` silently saturates to `math.MaxInt64`.
`handleBacktest` (`internal/api/api.go` ~line 171) only checks
`startingBalance.IsPositive()`, which a saturated `+Inf` balance passes
trivially. A request body `{"balance":"Inf", ...}` is accepted as a valid
~92-quadrillion-real balance and fed into `Backtest.Run`, risking silent
`int64` overflow in `go-money`'s `Multiply`/`Add` during quantity sizing
and profit accounting. Same issue via the CLI's `-balance` flag.

**Fix:** reject non-finite floats explicitly in `ParseMoney`, and/or
enforce a sane upper bound on balance.

## 3. [Low] Dead/orphaned file: `internal/strategies/bter.go`

`Holding`/`NewHolding` is not registered in `availableStrategies`
(`internal/strategies/base.go`), doesn't implement `domain.Strategy`, has
zero test coverage, and uses a Yahoo-Finance-style `.SA` ticker suffix
(`"BBAS3.SA"`) inconsistent with the rest of the codebase's bare B3 ticker
convention (`"PETR4"`). Looks like an accidentally committed scratch file.

**Fix:** delete it, or wire it up properly if it's meant to stay.

## 4. [Low] `PercentStopLoss` accepts `Percent >= 100`, becomes a silent permanent no-op

`internal/stoploss/base.go` lines 30-35 only validates `value <= 0`. With
`Percent >= 100`, `PercentStopLoss.triggerPrice`
(`internal/stoploss/percent.go` lines 25-29) produces a zero/negative
trigger price; since a candle's `Low` can never be negative, `Check`'s
guard (`candle.Low.Amount() > trigger.Amount()`) is always true, so the
stop-loss silently never fires.

**Fix:** validate `0 < value < 100` for the `"percent"` type.

## 5. [Low] Quantity sizing ignores stranded "dust" cash between trades

`internal/backtest/backtest.go`, `traverse` (lines 121-159). Each buy's
quantity is `balance.Amount() / order.Price.Amount()` (integer division),
leaving a fractional remainder uninvested; on close, `balance =
exit.Total()` (line 154) discards any earlier round's stranded change from
the pool used to size the *next* buy. Verified this does **not** corrupt
the final reported `Result.EndingBalance`/`Profit` (computed independently
in `compileResult`), but it does mean `traverse`'s own reinvestment pool
systematically undersizes later trades relative to true available capital
for frequently-trading strategies (e.g. `TwoCandleBreakout`). Not
documented in `traverse`'s doc comment, no test coverage of the behavior.

**Fix:** either document the behavior explicitly, or track accumulated
idle cash and include it when sizing the next buy.

## 6. [Low] Frontend has no automated tests

`frontend/package.json` has no test runner configured (no
`vitest`/`jest`/testing-library), and there isn't a single
`*.test.*`/`*.spec.*` file under `frontend/src`, despite non-trivial
client-side logic (`StrategyComparison.tsx`'s `stopLossValid`/`canSubmit`,
lines 182-193).

**Fix:** add `vitest` + `@testing-library/react`; start with the
`stopLossValid`/`canSubmit` logic and the asset-list tab/collapse state.

## 7. [Low] `internal/strategies` package has no direct unit tests for its registry

`go test -cover` shows `AvailableStrategyNamesList`,
`AvailableStrategyNames`, `LoadStrategy` (`internal/strategies/base.go`)
and both `BuyAndHold.Name()`/`TwoCandleBreakout.Name()` at 0% coverage
within the package itself — only exercised indirectly through
`internal/api`'s HTTP tests.

**Fix:** add direct unit tests for the registry functions and `Name()`
methods.

## 8. [Style] Dead/duplicated Tailwind class string

`frontend/src/styles/ui.ts`'s exported `cardClasses` (lines 47-49) is
never imported anywhere; `BacktestResultCard.tsx` independently
reimplements a near-identical string locally (lines 17-23) instead of
reusing it — the exact drift `ui.ts`'s own top-of-file comment says it
exists to prevent.

**Fix:** have `BacktestResultCard.tsx` import and reuse `cardClasses`, or
delete it if it's genuinely obsolete.

## 9. [Style] CLI didn't follow the Ticker→Asset rename

`cmd/cli.go` still names its flags/locals `-ticker`/`ticker` (lines 17,
91, 96-98) even though `domain.Backtest.Asset`, the HTTP API's `asset`
field, and `openapi.yaml` completed the rename. May be intentional (only
domain/API were in scope per `docs/plans/ticker-to-asset.md`), but worth a
deliberate decision rather than an oversight.

**Fix:** rename the CLI flag/locals to `asset` for consistency, unless
there's a reason to keep `-ticker` as the user-facing flag name.

## 10. [Style] Duplicated initial-balance magic number

`frontend/src/App.tsx` hardcodes `"10000.00"` for its own `balance` state
(line 12) and `10000` again as `CurrencyInput`'s `initialValue` (line 28),
with no shared constant. Currently harmless (the mount effect immediately
overwrites `App`'s state with the same value), but two independent sources
of truth that could silently diverge on a future edit.

**Fix:** extract a single `DEFAULT_BALANCE` constant used by both.

## What's solid (no action needed)

- Backend Go test suite is comprehensive and passes (`go test ./...`),
  including direct coverage of the OCO tie-break rule
  (`internal/backtest/backtest_test.go`'s
  `TestTraverse_TieBreak_StopLossWinsOverStrategyExit` and `pickExit` unit
  tests) and 100% coverage on `internal/stoploss`.
- No secrets, credentials, or hardcoded tokens found in the tree.
- `domain.Operation.Profit`/`Outcome`, `Order.Total`, and `Position.Close`
  are small, well-documented, and correctly tested.
- Frontend's `useAssetInfo`/`useAssetList`/`StrategyComparison` hooks
  correctly use `cancelled` flags to guard against stale-response state
  updates on unmount/re-fetch.
- `openapi.yaml` is thorough and largely matches the Go handler
  implementation (status codes, error shapes, verbose/omitempty behavior).

## Suggested order of work

1. Fix #1 (path traversal) and #2 (Inf/NaN balance) — the only two
   findings with real security/correctness impact.
2. Delete #3 (`bter.go`) — trivial cleanup.
3. Fix #4 (percent stop-loss validation) — small, low-risk.
4. Decide on #9 (CLI naming) — quick decision, then rename or leave as-is
   deliberately.
5. #5, #6, #7, #8, #10 are lower priority; batch as follow-up cleanup work
   whenever convenient.
