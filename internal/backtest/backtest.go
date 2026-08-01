package backtest

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/cotahist"
	"github.com/lucasbz/backtests/internal/domain"
)

// Backtest is the configuration for a single backtest run: which asset,
// over what timeframe, starting from how much cash, using which strategy,
// and optionally which stop-loss to race against the strategy's own exit
// signal.
type Backtest struct {
	// Asset is the ticker symbol to backtest, e.g. "PETR4".
	Asset    string
	Start    time.Time
	End      time.Time
	Balance  money.Money
	Strategy domain.Strategy
	// StopLoss is optional risk control, raced against Strategy.Decide each
	// candle a position is open: whichever fires first closes the
	// position, with StopLoss winning ties (see pickExit). nil means no
	// stop-loss - existing callers get exactly the behavior they had before
	// StopLoss existed.
	StopLoss domain.StopLoss
}

// Result summarizes the outcome of a Backtest.Run: every operation the
// strategy produced, and how the balance moved because of them.
type Result struct {
	StrategyName    string
	Operations      []domain.Operation
	StartingBalance money.Money
	EndingBalance   money.Money
	Profit          money.Money
	TotalOperations int
	Gains           int
	Losses          int

	// ProfitPercentage is Profit relative to StartingBalance (e.g. 12.5 for
	// a 12.5% return). Display-only, like money.Money.AsMajorUnits.
	ProfitPercentage float64
	// WinRate is the share of operations that were gains, e.g. 66.67 for 2
	// out of 3.
	WinRate float64
}

// resultJSON mirrors Result but with money.Money fields converted to plain
// JSON numbers (see domain.Candle's candleJSON for the same pattern), and
// Operations marked omitempty so API callers can drop it (e.g. when the
// caller didn't ask for verbose output) by nil-ing it out before marshaling.
type resultJSON struct {
	StrategyName     string             `json:"strategyName"`
	StartingBalance  float64            `json:"startingBalance"`
	EndingBalance    float64            `json:"endingBalance"`
	Profit           float64            `json:"profit"`
	ProfitPercentage float64            `json:"profitPercentage"`
	TotalOperations  int                `json:"totalOperations"`
	Gains            int                `json:"gains"`
	Losses           int                `json:"losses"`
	WinRate          float64            `json:"winRate"`
	Operations       []domain.Operation `json:"operations,omitempty"`
}

func (r Result) MarshalJSON() ([]byte, error) {
	return json.Marshal(resultJSON{
		StrategyName:     r.StrategyName,
		StartingBalance:  r.StartingBalance.AsMajorUnits(),
		EndingBalance:    r.EndingBalance.AsMajorUnits(),
		Profit:           r.Profit.AsMajorUnits(),
		ProfitPercentage: r.ProfitPercentage,
		TotalOperations:  r.TotalOperations,
		Gains:            r.Gains,
		Losses:           r.Losses,
		WinRate:          r.WinRate,
		Operations:       r.Operations,
	})
}

// Run loads b.Asset's candles within [b.Start, b.End], in date order, then
// drives them one at a time through traverse (which owns the candle loop,
// the running available cash, and the open Position - see traverse), and
// compiles the resulting operations into a Result.
func (b *Backtest) Run() (*Result, error) {
	candles, err := cotahist.LoadCandles(b.Asset, b.Start, b.End)
	if err != nil {
		return nil, fmt.Errorf("loading candles for %s: %w", b.Asset, err)
	}

	operations, err := traverse(b.Strategy, b.StopLoss, b.Balance, candles)
	if err != nil {
		return nil, err
	}

	return compileResult(b.Strategy.Name(), operations, b.Balance)
}

// traverse feeds candles to strategy one at a time, in order, tracking the
// running available cash and any currently open domain.Position:
//
//   - While flat, it asks strategy to Decide whether to buy. If it does,
//     the buy's quantity is sized from the running available cash and the
//     proposed entry price (available / price); if that comes out to zero
//     or less (available cash can't afford even one share), the buy is
//     skipped and it stays flat. Quantity sizing is integer division, so a
//     buy almost always leaves a fractional remainder of available
//     unspent ("dust"); that remainder stays in available rather than
//     being discarded, so it compounds into later buys instead of being
//     silently lost each cycle.
//   - While holding, it asks strategy to Decide whether to exit, and, if
//     stopLoss is set, also asks stopLoss to Check the same candle - an OCO
//     race between the strategy's own signal and the risk control. Whoever
//     fires wins (see pickExit); the position closes into a completed
//     domain.Operation, and its full sale proceeds are added to whatever
//     was already sitting in available (not used to replace it), so the
//     next buy is sized off true total cash on hand - starting balance,
//     plus every prior cycle's stranded dust, plus this sale's proceeds -
//     not just this round's proceeds in isolation.
//
// Any Position still open when candles run out is intentionally dropped -
// it's up to strategy's own Decide to force-close before the last candle
// (via the isLast argument) if it wants every position to close by the end
// of the run; see domain.Strategy.
func traverse(strategy domain.Strategy, stopLoss domain.StopLoss, startingBalance money.Money, candles []domain.Candle) ([]domain.Operation, error) {
	var position *domain.Position
	var operations []domain.Operation
	available := startingBalance

	for i, candle := range candles {
		isLast := i == len(candles)-1

		if position == nil {
			order := strategy.Decide(candle, nil, isLast)
			if order == nil {
				continue
			}
			order.Quantity = available.Amount() / order.Price.Amount()
			if order.Quantity <= 0 {
				continue // can't afford even one share at this price yet
			}

			spent := order.Total()
			leftover, err := available.Subtract(&spent)
			if err != nil {
				return nil, fmt.Errorf("computing leftover cash after buy on %s: %w", order.Date, err)
			}
			available = *leftover

			position = &domain.Position{Buy: *order}
			continue
		}

		strategyExit := strategy.Decide(candle, position, isLast)
		var stopExit *domain.Order
		if stopLoss != nil {
			stopExit = stopLoss.Check(candle, *position)
		}

		exit := pickExit(strategyExit, stopExit)
		if exit == nil {
			continue
		}
		exit.Quantity = position.Buy.Quantity
		operations = append(operations, position.Close(*exit))

		proceeds := exit.Total()
		newAvailable, err := available.Add(&proceeds)
		if err != nil {
			return nil, fmt.Errorf("accumulating sale proceeds on %s: %w", exit.Date, err)
		}
		available = *newAvailable

		position = nil
	}

	return operations, nil
}

// pickExit resolves the OCO race between the strategy's own exit signal and
// the stop-loss's when a position is open: if both fire on the same
// candle, the stop-loss wins - it's the risk control, and this matches how
// a real OCO stop order is generally assumed to fill first under adverse
// price action (see docs/plans/stop-loss-oco.md's "Tie-break rule"). If
// only one of them fires, that one wins; if neither does, nil (no exit this
// candle).
func pickExit(strategyExit, stopExit *domain.Order) *domain.Order {
	if stopExit != nil {
		return stopExit
	}
	return strategyExit
}

// compileResult turns a completed operations list into a Result, given the
// strategy's display name and the backtest's starting balance: go over each
// operation, adding its gain or loss to the running total profit and
// balance, and counting it as a gain or a loss.
func compileResult(strategyName string, operations []domain.Operation, startingBalance money.Money) (*Result, error) {
	total := len(operations)

	profit := money.New(0, domain.Currency)
	balance := startingBalance
	gains, losses := 0, 0
	for _, op := range operations {
		outcome, err := op.Outcome()
		if err != nil {
			return nil, fmt.Errorf("computing outcome for operation on %s: %w", op.Date, err)
		}
		switch outcome {
		case domain.Gain:
			gains++
		case domain.Loss:
			losses++
		}

		opProfit, err := op.Profit()
		if err != nil {
			return nil, fmt.Errorf("computing profit for operation on %s: %w", op.Date, err)
		}

		profit, err = profit.Add(&opProfit)
		if err != nil {
			return nil, fmt.Errorf("accumulating profit: %w", err)
		}

		newBalance, err := balance.Add(&opProfit)
		if err != nil {
			return nil, fmt.Errorf("updating balance: %w", err)
		}
		balance = *newBalance
	}

	var profitPercentage float64
	if !startingBalance.IsZero() {
		profitPercentage = profit.AsMajorUnits() / startingBalance.AsMajorUnits() * 100
	}

	var winRate float64
	if total > 0 {
		winRate = float64(gains) / float64(total) * 100
	}

	return &Result{
		StrategyName:     strategyName,
		Operations:       operations,
		StartingBalance:  startingBalance,
		EndingBalance:    balance,
		Profit:           *profit,
		TotalOperations:  total,
		Gains:            gains,
		Losses:           losses,
		ProfitPercentage: profitPercentage,
		WinRate:          winRate,
	}, nil
}
