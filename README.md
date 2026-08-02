# backtests

A command-line backtester for stocks traded on B3 (the Brazilian stock
exchange), driven by B3's own historical price files (COTAHIST).

## Requirements

- Go 1.25+
- A B3 COTAHIST annual file for each year you want to backtest, e.g.
  `COTAHIST_A2015.TXT`, downloaded from B3's market data site and placed in
  `resources/`.

## 1. Import price data

COTAHIST files are B3's raw fixed-width format. Import one year at a time
into per-ticker JSON files under `resources/cotahist/<TICKER>/<TICKER>_<YEAR>.json`:

```sh
make import IMPORT_YEAR=2015
```

which runs:

```sh
go run scripts/import_cotahist.go -in resources/COTAHIST_A2015.TXT -out resources/cotahist
```

Re-running the import for a year you already imported overwrites just that
year's files — other years are left untouched, so you can import multiple
years incrementally.

## 2. Look up available data for a ticker

Before backtesting, check what date range is actually imported for a ticker:

```sh
make info ASSET=PETR4
```

which runs:

```sh
go run ./cmd info -asset PETR4
```

```
PETR4: data available from 2010-01-04 to 2026-07-30
```

## 3. Run a backtest

```sh
make backtest ASSET=PETR4 START=2015-01-02 END=2015-12-30 STRATEGY=buy-and-hold BALANCE=10000.00
```

which runs:

```sh
go run ./cmd backtest -asset PETR4 -start 2015-01-02 -end 2015-12-30 -strategy buy-and-hold -balance 10000.00
```

All of `ASSET`, `START`, `END`, `STRATEGY` and `BALANCE` have defaults in
the `Makefile`, so `make backtest` alone will run with those.

Flags:

| Flag | Description |
|---|---|
| `-asset` | Ticker of the asset to backtest, e.g. `PETR4`. Must already be imported (step 1). |
| `-start` / `-end` | Inclusive date range, `YYYY-MM-DD`. Only trading days have data — B3 holidays and weekends have no quotes. |
| `-strategy` | Which strategy to run (see below). |
| `-balance` | Starting cash balance, e.g. `10000.00`. Position sizing spends as much of this as affords whole shares at the buy price. |
| `-v` | Optional. Print each BUY/SELL operation (see below). Off by default. |
| `-config` | Optional. Path to a JSON file providing defaults for `-asset`, `-start`, `-end`, `-balance`, `-strategy`, `-v`, and `strategyParams` (see below). Flags passed explicitly always override the config file. |

### Using a config file

Instead of typing every flag, put the common ones in a JSON file and point
`-config` at it:

```json
{
  "asset": "PETR4",
  "start": "2015-01-02",
  "end": "2015-12-30",
  "balance": "10000.00",
  "strategy": "buy-and-hold",
  "verbose": false
}
```

```sh
go run ./cmd backtest -config backtest.json
```

is equivalent to:

```sh
go run ./cmd backtest -asset PETR4 -start 2015-01-02 -end 2015-12-30 -balance 10000.00 -strategy buy-and-hold
```

All fields in the config file are optional, and any flag passed explicitly
on the command line overrides that field from the file — the file only
fills in defaults for flags you didn't pass. `compare` supports the same
`-config` flag, with `strategy` in the file meaning the challenger strategy
(same restriction as the `-strategy` flag: it can't be `buy-and-hold`).

`strategyParams` is an object of named numeric params for whichever
strategy you picked (see [Available strategies](#available-strategies)
below for what each parameterized strategy expects) — it has no
per-key CLI flag, so it's only settable via this config file:

```json
{
  "asset": "PETR4",
  "start": "2015-01-02",
  "end": "2015-12-30",
  "balance": "10000.00",
  "strategy": "sma-crossover",
  "strategyParams": { "shortPeriod": 5, "longPeriod": 20 }
}
```

### Available strategies

- `buy-and-hold` — buys once, at the low of the first candle in the
  timeframe, and sells once, at the high of the last candle in the timeframe.
- `two-candle-breakout` — on every candle, looks at the low/high of the two
  candles before it: buys once price reaches the minimum of those two lows,
  and (once holding) sells once price reaches the maximum of those two
  highs. Repeatedly buys and sells as new two-candle windows trigger; see
  `internal/strategies/two_candle_breakout.go` for the exact rule.
- `sma-crossover` / `ema-crossover` — track a "short" and a "long" moving
  average (Simple/Exponential respectively), configured via
  `strategyParams.shortPeriod`/`strategyParams.longPeriod`; buys, while
  flat, when the short average crosses above the long one, and sells,
  while holding, when it crosses back below - both at that candle's
  closing price. See `internal/strategies/crossover.go`.
- `rsi-threshold` — tracks a Relative Strength Index (RSI) over
  `strategyParams.period` candles; buys, while flat, when RSI crosses
  below `strategyParams.oversold`, and sells, while holding, when it
  crosses above `strategyParams.overbought` - both at that candle's
  closing price. See `internal/strategies/rsi_threshold.go`.
- `ema-trend-breakout` — the same two-candle-window buy/sell trigger as
  `two-candle-breakout`, but the buy side additionally requires an EMA
  uptrend: configured via `strategyParams.shortPeriod`/
  `strategyParams.longPeriod` (defaults 8/80), it only buys when the short
  EMA is above the long EMA and the current close is above the short EMA;
  the sell side is unchanged. See `internal/strategies/ema_trend_breakout.go`.

New strategies are added by implementing `domain.Strategy`
(`internal/domain/strategy.go`) and registering a constructor in
`internal/strategies/base.go`.

### Reading the output

The result is always a single summary line. With `-v`, a table of every
operation is also printed below it (mirroring the operations table in the
web frontend):

```
Running Backtest for: PETR4 2015-01-02 to 2015-12-30 | Strategy: Buy & Hold | Balance: R$10.000,00 -> R$7.223,20 (Profit: -R$2.776,80, -27.77%) | G/L/T: 0/1/1 (WR: 0.00%) | Max DD: R$2.776,80 (27.77%)
Operations (1):
#  Buy date    Buy price  Sell date   Sell price  Qty   Profit
1  2015-01-02  R$9,36     2015-12-30  R$6,76      1068  -R$2.776,80
```

Without `-v`, only the summary line is printed.

- Each row of the operations table (`-v` only) is one operation the
  strategy made.
- `G/L/T` is Gains/Losses/Total operations that closed with a profit or a
  loss (breaking even counts as a gain).
- `Win rate` is `Gains / total operations`.
- `Max DD` (max drawdown) is the largest peak-to-trough decline in balance
  across the operations, shown both as a currency amount and as a percentage
  of the peak. It's `0.00%` if the balance never declined below a prior peak
  (including when there are no operations).

## 4. Compare a strategy against Buy & Hold

```sh
make compare ASSET=PETR4 START=2015-01-02 END=2015-12-30 COMPARE_STRATEGY=two-candle-breakout BALANCE=10000.00
```

which runs:

```sh
go run ./cmd compare -asset PETR4 -start 2015-01-02 -end 2015-12-30 -strategy two-candle-breakout -balance 10000.00
```

`compare` always runs Buy & Hold as a fixed baseline, plus a second,
user-chosen "challenger" strategy, over the same asset/timeframe/balance, and
prints both results side by side followed by a one-line summary of which one
won:

```
=== Baseline: Buy & Hold ===
Running Backtest for: PETR4 2015-01-02 to 2015-12-30 | Strategy: Buy & Hold | Balance: R$10.000,00 -> R$7.223,20 (Profit: -R$2.776,80, -27.77%) | G/L/T: 0/1/1 (WR: 0.00%) | Max DD: R$2.776,80 (27.77%)
=== Challenger: two-candle-breakout ===
Running Backtest for: PETR4 2015-01-02 to 2015-12-30 | Strategy: Two-Candle Breakout | Balance: R$10.000,00 -> R$10.842,10 (Profit: R$842,10, 8.42%) | G/L/T: 5/3/8 (WR: 62.50%) | Max DD: R$643,50 (6.15%)

Result: Challenger outperformed baseline by 36.19 percentage points
```

Flags are the same as `backtest` (`-asset`, `-start`, `-end`, `-balance`,
`-v`, `-config` — see [Using a config file](#using-a-config-file) above),
except `-strategy` picks the *challenger* only — Buy & Hold is always
the baseline, so it can't also be passed as `-strategy` (the command errors
out if you try). The comparison metric is `ProfitPercentage`, the same
percentage shown in each result's own summary line.

Note `make compare`/`make v-compare` read a separate `COMPARE_STRATEGY`
Makefile variable for the challenger, not `STRATEGY` — `STRATEGY` (default
`buy-and-hold`) is only used by `make backtest`/`make v-backtest`.
`COMPARE_STRATEGY` defaults to `two-candle-breakout`, so `make compare`
alone (no override) already picks a real challenger; override it with
`COMPARE_STRATEGY=<name>` to compare a different one.

## 5. Scan every asset for a strategy that beats Buy & Hold

```sh
make scan COMPARE_STRATEGY=two-candle-breakout START=2015-01-02 END=2015-12-30 BALANCE=10000.00 YEAR=2015
```

which runs:

```sh
go run ./cmd scan -strategy two-candle-breakout -start 2015-01-02 -end 2015-12-30 -balance 10000.00 -year 2015
```

`scan` is `compare`'s batch sibling: instead of comparing one asset against
Buy & Hold, it runs the same comparison across *every* imported asset
(optionally restricted to a year via `-year`), so you can see which assets
a strategy actually wins on rather than trying assets one at a time by hand:

```
Scanning 615 assets: two-candle-breakout vs Buy & Hold (2015-01-02 to 2015-12-30)...

Asset  Baseline%  Challenger%  Delta    Won
PETR4  -27.77     11.00        +38.77   yes
VALE3  8.10       8.10         +0.00    no
...

Won on 255/615 assets (41.46%)
```

Flags are the same as `compare` (`-start`, `-end`, `-balance`, `-strategy`,
`-v`, `-config`), minus `-asset` (a scan runs every asset, not one), plus:

| Flag | Description |
|---|---|
| `-year` | Optional. Restrict the scanned assets to those with imported data for this year, same filter as `GET /api/assets?year=`. `0` (default) scans every imported asset. |

`-v` additionally prints a full operations table (same shape as
`backtest`/`compare`'s `-v`) for every asset the challenger strategy
actually traded on - useful for drilling into a specific winner without
re-running `compare` by hand.

Like `compare`, the baseline is always Buy & Hold, so `-strategy` can't be
`buy-and-hold` either. Results are sorted with winners first (by `Delta`
descending); any asset whose backtest failed to load/run is listed
separately at the end, under an `Errors:` section, rather than aborting the
whole scan.

Note `make scan`/`make v-scan` also read `COMPARE_STRATEGY` (not
`STRATEGY`), same as `compare`/`v-compare` above (see that section's note
on `COMPARE_STRATEGY` vs `STRATEGY`). `make v-scan` is `make scan` with
`-v` (per-asset operations tables) turned on.

`scan` is also available as `POST /api/scan` - see
[openapi.yaml](openapi.yaml).

## HTTP API

`backtest` and `info` are also available as a JSON HTTP API:

```sh
make serve
```

which runs:

```sh
go run ./cmd serve -addr :8080
```

See [openapi.yaml](openapi.yaml) for the full contract (endpoints,
request/response shapes, error format).

## Frontend

A React UI for the HTTP API lives in `frontend/`. Start its dev server with:

```sh
make dev
```

which runs:

```sh
cd frontend && npm run dev
```

## Running tests

```sh
make test
```

which runs `go test ./...`.

## Project layout

- `scripts/import_cotahist.go` — parses COTAHIST files into per-ticker JSON.
- `internal/domain` — core types: `Candle`, `Order`, `Operation`, `Position`,
  `Strategy`, `StopLoss`.
- `internal/cotahist` — loads imported candle JSON back into `domain.Candle`s,
  and looks up a ticker's available date range.
- `internal/strategies` — concrete `domain.Strategy` implementations and the
  `-strategy` name registry. Each `Strategy.Decide` is called once per
  candle by `Backtest`, which owns the candle loop, the running balance, and
  the currently open `Position`. `sma-crossover`/`ema-crossover`/
  `rsi-threshold` compose `internal/indicators` implementations rather than
  computing their own rolling state.
- `internal/indicators` — reusable technical indicators (`SMA`, `EMA`,
  `RSI`) fed one candle at a time, plus their own name registry, shared
  across whichever `internal/strategies` implementations need them.
- `internal/stoploss` — concrete `domain.StopLoss` implementations (optional
  risk control raced against a strategy's own exit signal - see
  [openapi.yaml](openapi.yaml)'s `stopLoss` request field) and their own
  name registry, mirroring `internal/strategies`.
- `internal/backtest` — `Backtest.Run()`: drives a strategy (and, if
  configured, a stop-loss) through a ticker's candles and compiles the
  resulting `Result` (profit, balance, gains/losses, win rate). `Scan()`
  runs the same comparison `compare`/`scan` do across every asset
  concurrently, loading a fresh strategy instance per asset (strategies
  carry mutable per-run state, so one instance can never safely back more
  than one asset's backtest).
- `internal/api` — JSON HTTP handlers wrapping the same operations as the CLI
  (see [openapi.yaml](openapi.yaml)).
- `cmd/main.go` — the CLI entrypoint described above, plus the `serve`
  subcommand that starts the HTTP API.
