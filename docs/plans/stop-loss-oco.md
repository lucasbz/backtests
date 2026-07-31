# Stop-loss (OCO) design plan

## Goal

Add stop-loss risk control on top of any strategy's own exit signal, structured
as an OCO (one-cancels-the-other) race: once a position is open, whichever
exit condition is touched first by a candle — the strategy's own signal, or
the stop-loss's — closes the position.

Stop-loss rules come in variants (fixed %, fixed currency amount, later
ATR/volatility-based). Each variant is its own pluggable implementation,
mirroring how strategies are pluggable today.

## Current architecture (for reference)

- `domain.Strategy` interface: `Traverse(candle Candle)`, `Name() string`,
  `Operations() []Operation`.
- Each strategy accumulates candles internally via `Traverse`, then computes
  the full list of completed buy/sell `Operation`s in one pass inside
  `Operations()`, called once after all candles have been traversed.
- Each strategy holds its own "am I currently in a position" state as loose
  local variables (`holding bool`, `buyOrder domain.Order`) private to that
  loop.
- Quantity sizing (`balance / entryPrice`) is computed independently inside
  each strategy that needs it — duplicated logic.
- `Backtest.Run()` loads candles and calls into the strategy; the strategy
  owns the whole traversal.

This works but has no seam for a second, independent thing (stop-loss) to
observe the position candle-by-candle and race the strategy's own exit —
`Operations()` only sees the final result, after the fact.

## New architecture

### `domain.Position`

A small type representing a currently open trade, owned by `Backtest`
(not by the strategy):

```go
type Position struct {
    Buy Order
}

func (p Position) Close(sell Order) Operation {
    return Operation{Date: p.Buy.Date, BuyOrder: p.Buy, SellOrder: sell}
}
```

### `Strategy` interface (revised)

`Backtest` now drives the candle loop and owns the `Position`. Each candle,
it asks the strategy for a decision, passing whether a position is currently
open:

```go
type Strategy interface {
    Name() string
    // Decide is called once per candle, in chronological order. `position`
    // is the currently open position, or nil if flat. Returns the order to
    // place this candle (Buy if position is nil, Sell if not), or nil if no
    // action should be taken this candle.
    //
    // Buy orders returned here have Quantity 0 - Backtest fills it in from
    // the running balance and the proposed entry price. Sell orders should
    // set Quantity to position.Buy.Quantity.
    Decide(candle Candle, position *Position) *Order
}
```

Strategies keep whatever rolling window of recent candles they need
internally (e.g. two-candle-breakout still tracks the last two candles),
but no longer hold "am I in a position" state or build `Operations()`
themselves — `Backtest` does both.

A strategy that wants to guarantee its position closes by the end of the
backtest (e.g. Buy & Hold) does so simply by treating "this is the last
candle" as an exit signal inside its own `Decide` — no special hook needed.
A strategy that's fine dropping an unclosed position at the end (e.g.
two-candle-breakout, per existing documented/tested behavior) just never
returns such a signal, and `Backtest` discards the dangling open `Position`
when the loop ends. This preserves both strategies' current, already-tested
behavior with no special-casing in `Backtest`.

### `StopLoss` interface (new)

A sibling concept to `Strategy`, only ever consulted while a position is
open:

```go
type StopLoss interface {
    Name() string
    // Check is called once per candle while a position is open. Returns the
    // sell order to place if the stop-loss condition is triggered this
    // candle, or nil otherwise.
    Check(candle Candle, position Position) *Order
}
```

Initial implementations, each its own type (mirroring the `Strategy`
registry pattern in `internal/strategies/base.go`):

- `PercentStopLoss` — trigger price = entry price × (1 − percent).
- `FixedAmountStopLoss` — trigger price = entry price − fixed amount.
- (Future) `ATRStopLoss` — trigger price adapts to recent volatility. Not
  built now; would need its own rolling-window state (similar to how
  strategies track recent candles), which the interface above already
  supports without changes — `Check` receiving one candle at a time is
  enough to let an ATR implementation maintain internal history the same
  way strategies do.

A registry (`availableStopLosses`, same shape as `availableStrategies`) maps
a name to a constructor. No stop-loss selected = current behavior, unchanged.

### `Backtest` loop (revised)

`Backtest` now owns the traversal, the running balance, the open
`Position`, and the completed `Operation` list:

```go
var position *Position
var operations []Operation
balance := bt.Balance

for _, candle := range candles {
    if position == nil {
        order := bt.Strategy.Decide(candle, nil)
        if order == nil {
            continue
        }
        order.Quantity = balance.Amount() / order.Price.Amount()
        if order.Quantity <= 0 {
            continue
        }
        position = &Position{Buy: *order}
        continue
    }

    strategyExit := bt.Strategy.Decide(candle, position)
    var stopExit *Order
    if bt.StopLoss != nil {
        stopExit = bt.StopLoss.Check(candle, *position)
    }

    exit := pickExit(strategyExit, stopExit) // stop-loss wins on a same-candle tie
    if exit == nil {
        continue
    }
    exit.Quantity = position.Buy.Quantity
    operations = append(operations, position.Close(*exit))
    balance = exit.Total()
    position = nil
}
// any still-open `position` here is intentionally dropped, per-strategy
// behavior as described above
```

`bt.StopLoss` is optional (nil = no stop-loss, existing behavior exactly
preserved for callers that don't ask for one).

### Tie-break rule

If both the strategy's own exit signal and the stop-loss fire on the same
candle, **stop-loss wins** — it's the risk control, and this matches how a
real OCO stop order is generally assumed to fill first under adverse price
action. Flagging this as the chosen default; easy to revisit if it turns out
to matter for a specific test case.

### API changes

`POST /api/backtest` gains an optional stop-loss selector, e.g.:

```json
{
  "ticker": "PETR4",
  "start": "2020-01-01",
  "end": "2020-12-31",
  "balance": "10000.00",
  "strategy": "two-candle-breakout",
  "stopLoss": { "type": "percent", "value": 5 }
}
```

Omitted `stopLoss` = current behavior, unchanged. `GET /api/strategies`
gets a sibling endpoint or field listing available stop-loss types (TBD at
implementation time — small addition, not architecturally significant).

## Implementation steps

1. Add `domain.Position` (+ tests).
2. Revise `domain.Strategy` interface to the `Decide(candle, position) *Order`
   shape; update `two_candle_breakout.go` and `buy_and_hold.go` to match,
   preserving their existing tested behavior (including the "drop unclosed
   position" and "close at last candle" semantics respectively). Update/port
   existing strategy tests to the new interface.
3. Add `domain.StopLoss` interface, `PercentStopLoss`, `FixedAmountStopLoss`
   (+ tests), and an `availableStopLosses` registry analogous to
   `availableStrategies`.
4. Move the candle-traversal loop, `Position` lifecycle, balance tracking,
   and quantity sizing into `Backtest.Run()` / `compileResult`, replacing
   each strategy's own `Operations()` computation. Remove `Operations()`
   from the `Strategy` interface.
5. Wire an optional `StopLoss` through `Backtest`, `internal/api/api.go`'s
   backtest request/response, and `openapi.yaml`.
6. Frontend: expose a stop-loss selector in `StrategyComparison.tsx` once
   the API supports it (separate follow-up, not part of this backend plan).

## Open questions / decisions made by default (flag if you disagree)

- Stop-loss wins ties on a same-candle simultaneous trigger.
- Quantity sizing moves from each strategy into `Backtest`, applied
  uniformly (dedupes existing duplicated logic, no behavior change for
  existing strategies).
- `StopLoss` has no entry-side role — it only ever evaluates once a
  position is already open.
