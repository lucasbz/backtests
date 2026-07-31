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
make info TICKER=PETR4
```

which runs:

```sh
go run ./cmd info -ticker PETR4
```

```
PETR4: data available from 2010-01-04 to 2026-07-30
```

## 3. Run a backtest

```sh
make backtest TICKER=PETR4 START=2015-01-02 END=2015-12-30 STRATEGY=buy-and-hold BALANCE=10000.00
```

which runs:

```sh
go run ./cmd backtest -ticker PETR4 -start 2015-01-02 -end 2015-12-30 -strategy buy-and-hold -balance 10000.00
```

All of `TICKER`, `START`, `END`, `STRATEGY` and `BALANCE` have defaults in
the `Makefile`, so `make backtest` alone will run with those.

Flags:

| Flag | Description |
|---|---|
| `-ticker` | Ticker to backtest, e.g. `PETR4`. Must already be imported (step 1). |
| `-start` / `-end` | Inclusive date range, `YYYY-MM-DD`. Only trading days have data — B3 holidays and weekends have no quotes. |
| `-strategy` | Which strategy to run. Currently only `buy-and-hold` (see below). |
| `-balance` | Starting cash balance, e.g. `10000.00`. Position sizing spends as much of this as affords whole shares at the buy price. |
| `-v` | Optional. Print each BUY/SELL operation (see below). Off by default. |

### Available strategies

- `buy-and-hold` — buys once, at the low of the first candle in the
  timeframe, and sells once, at the high of the last candle in the timeframe.

New strategies are added by implementing `domain.Strategy`
(`internal/domain/strategy.go`) and registering a constructor in
`internal/strategies/base.go`.

### Reading the output

The result is always a single summary line. With `-v`, each operation is
also printed above it:

```
  BUY  2015-01-02 @ R$9,36 x1068
  SELL 2015-12-30 @ R$6,76 x1068
Running Backtest for: PETR4 2015-01-02 to 2015-12-30 | Strategy: Buy & Hold | Balance: R$10.000,00 -> R$7.223,20 (Profit: -R$2.776,80, -27.77%) | Gains: 0 Losses: 1 (Win rate: 0.00%)
```

Without `-v`, only the summary line is printed.

- Each `BUY`/`SELL` line (`-v` only) is one operation the strategy made.
- `Gains`/`Losses` count operations that closed with a profit or a loss
  (breaking even counts as a gain).
- `Win rate` is `Gains / total operations`.

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
- `internal/domain` — core types: `Candle`, `Order`, `Operation`, `Strategy`.
- `internal/cotahist` — loads imported candle JSON back into `domain.Candle`s,
  and looks up a ticker's available date range.
- `internal/strategies` — concrete `domain.Strategy` implementations and the
  `-strategy` name registry.
- `internal/backtest` — `Backtest.Run()`: feeds a strategy its candles and
  compiles the resulting `Result` (profit, balance, gains/losses, win rate).
- `internal/api` — JSON HTTP handlers wrapping the same operations as the CLI
  (see [openapi.yaml](openapi.yaml)).
- `cmd/main.go` — the CLI entrypoint described above, plus the `serve`
  subcommand that starts the HTTP API.
