package backtest

import (
	"encoding/json"
	"fmt"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

type BacktestResult struct {
	StrategyName    string
	Operations      []domain.Operation
	StartingBalance money.Money
	EndingBalance   money.Money
	Profit          money.Money
	TotalOperations int
	Gains           int
	Losses          int

	// maxDrawdownPercentage and maxDrawdownAmount are computed once, in
	// NewBacktestResult, from Operations - which callers may clear afterward to
	// trim response payloads (see internal/api's handleBacktest nil-ing
	// Operations for non-verbose requests). Caching them here rather than
	// recomputing from Operations on every MaxDrawdownPercentage/
	// MaxDrawdownAmount call keeps them correct regardless of whether
	// Operations is still populated by the time they're called.
	//
	// Both are unexported specifically so a BacktestResult can only be
	// built through NewBacktestResult: money.Money's zero value isn't safe
	// to display, so an exported field here could be left unset by a
	// struct literal built outside this package and panic the first time
	// it's read (this happened once already, in internal/cli's tests,
	// before NewBacktestResult existed) - keeping the fields private
	// closes that off entirely.
	maxDrawdownPercentage float64
	maxDrawdownAmount     money.Money
}

// ProfitPercentage is Profit relative to StartingBalance (e.g. 12.5 for a
// 12.5% return). Zero when StartingBalance is zero.
func (r BacktestResult) ProfitPercentage() float64 {
	if r.StartingBalance.IsZero() {
		return 0
	}
	return r.Profit.AsMajorUnits() / r.StartingBalance.AsMajorUnits() * 100
}

// WinRate is the share of operations that were gains, e.g. 66.67 for 2 out
// of 3. Zero when there were no operations.
func (r BacktestResult) WinRate() float64 {
	if r.TotalOperations == 0 {
		return 0
	}
	return float64(r.Gains) / float64(r.TotalOperations) * 100
}

// MaxDrawdownPercentage is the largest peak-to-trough decline in balance
// across the operations that produced this BacktestResult, as a
// percentage of the peak (e.g. 15.23 for a 15.23% drawdown). Zero if the
// balance never declined below a prior peak (including when there were no
// operations).
func (r BacktestResult) MaxDrawdownPercentage() float64 {
	return r.maxDrawdownPercentage
}

// MaxDrawdownAmount is the largest peak-to-trough decline in balance
// across the operations that produced this BacktestResult, in absolute
// currency terms (always non-negative). Zero under the same conditions as
// MaxDrawdownPercentage.
func (r BacktestResult) MaxDrawdownAmount() money.Money {
	return r.maxDrawdownAmount
}

// backtestResultJSON mirrors BacktestResult but with money.Money fields
// converted to plain
// JSON numbers (see domain.Candle's candleJSON for the same pattern), and
// Operations marked omitempty so API callers can drop it (e.g. when the
// caller didn't ask for verbose output) by nil-ing it out before marshaling.
type backtestResultJSON struct {
	StrategyName          string             `json:"strategyName"`
	StartingBalance       float64            `json:"startingBalance"`
	EndingBalance         float64            `json:"endingBalance"`
	Profit                float64            `json:"profit"`
	ProfitPercentage      float64            `json:"profitPercentage"`
	TotalOperations       int                `json:"totalOperations"`
	Gains                 int                `json:"gains"`
	Losses                int                `json:"losses"`
	WinRate               float64            `json:"winRate"`
	MaxDrawdownPercentage float64            `json:"maxDrawdownPercentage"`
	MaxDrawdownAmount     float64            `json:"maxDrawdownAmount"`
	Operations            []domain.Operation `json:"operations,omitempty"`
}

func (r BacktestResult) MarshalJSON() ([]byte, error) {
	maxDrawdownAmount := r.MaxDrawdownAmount()
	return json.Marshal(backtestResultJSON{
		StrategyName:          r.StrategyName,
		StartingBalance:       r.StartingBalance.AsMajorUnits(),
		EndingBalance:         r.EndingBalance.AsMajorUnits(),
		Profit:                r.Profit.AsMajorUnits(),
		ProfitPercentage:      r.ProfitPercentage(),
		TotalOperations:       r.TotalOperations,
		Gains:                 r.Gains,
		Losses:                r.Losses,
		WinRate:               r.WinRate(),
		MaxDrawdownPercentage: r.MaxDrawdownPercentage(),
		MaxDrawdownAmount:     maxDrawdownAmount.AsMajorUnits(),
		Operations:            r.Operations,
	})
}

// NewBacktestResult turns a completed operations list into a
// BacktestResult, given the strategy's display name and the backtest's
// starting balance: go over each operation, adding its gain or loss to the
// running total profit and balance, counting it as a gain or a loss, and
// tracking the running peak balance to find the largest peak-to-trough
// decline (maxDrawdownPercentage/maxDrawdownAmount, cached on
// BacktestResult - see its MaxDrawdownPercentage/MaxDrawdownAmount
// methods - since Operations, which this walk depends on, may be cleared
// by callers after the fact to trim response payloads).
//
// This is the only way to construct a valid BacktestResult - both fields
// backing MaxDrawdownPercentage/MaxDrawdownAmount are unexported, so a
// BacktestResult built any other way (e.g. a struct literal from another
// package) can't set them and would report a zero drawdown regardless of
// its Operations. Callers that need a BacktestResult for a test, without
// going through a full Backtest.Run(), should still call
// NewBacktestResult with real Operations rather than fabricate one field
// at a time.
func NewBacktestResult(strategyName string, operations []domain.Operation, startingBalance money.Money) (*BacktestResult, error) {
	total := len(operations)

	profit := money.New(0, domain.Currency)
	balance := startingBalance
	gains, losses := 0, 0
	peak := startingBalance
	maxDrawdownAmount := *money.New(0, domain.Currency)
	var maxDrawdownPercentage float64
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

		isNewPeak, err := balance.GreaterThan(&peak)
		if err != nil {
			return nil, fmt.Errorf("comparing balance to peak on %s: %w", op.Date, err)
		}
		if isNewPeak {
			peak = balance
		} else if peak.AsMajorUnits() > 0 {
			drawdownAmount, err := peak.Subtract(&balance)
			if err != nil {
				return nil, fmt.Errorf("computing drawdown amount on %s: %w", op.Date, err)
			}
			drawdownPercentage := drawdownAmount.AsMajorUnits() / peak.AsMajorUnits() * 100
			if drawdownPercentage > maxDrawdownPercentage {
				maxDrawdownPercentage = drawdownPercentage
				maxDrawdownAmount = *drawdownAmount
			}
		}
	}

	return &BacktestResult{
		StrategyName:          strategyName,
		Operations:            operations,
		StartingBalance:       startingBalance,
		EndingBalance:         balance,
		Profit:                *profit,
		TotalOperations:       total,
		Gains:                 gains,
		Losses:                losses,
		maxDrawdownPercentage: maxDrawdownPercentage,
		maxDrawdownAmount:     maxDrawdownAmount,
	}, nil
}
