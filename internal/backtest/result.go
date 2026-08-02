package backtest

import (
	"encoding/json"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

type Result struct {
	StrategyName    string
	Operations      []domain.Operation
	StartingBalance money.Money
	EndingBalance   money.Money
	Profit          money.Money
	TotalOperations int
	Gains           int
	Losses          int

	// maxDrawdownPercentage and maxDrawdownAmount are computed once, in
	// NewResult, from Operations - which callers may clear afterward to
	// trim response payloads (see internal/api's handleBacktest nil-ing
	// Operations for non-verbose requests). Caching them here rather than
	// recomputing from Operations on every MaxDrawdownPercentage/
	// MaxDrawdownAmount call keeps them correct regardless of whether
	// Operations is still populated by the time they're called.
	//
	// Both are unexported specifically so a Result can only be built
	// through NewResult: money.Money's zero value isn't safe to display,
	// so an exported field here could be left unset by a struct literal
	// built outside this package and panic the first time it's read (this
	// happened once already, in internal/cli's tests, before NewResult
	// existed) - keeping the fields private closes that off entirely.
	maxDrawdownPercentage float64
	maxDrawdownAmount     money.Money
}

// ProfitPercentage is Profit relative to StartingBalance (e.g. 12.5 for a
// 12.5% return). Zero when StartingBalance is zero.
func (r Result) ProfitPercentage() float64 {
	if r.StartingBalance.IsZero() {
		return 0
	}
	return r.Profit.AsMajorUnits() / r.StartingBalance.AsMajorUnits() * 100
}

// WinRate is the share of operations that were gains, e.g. 66.67 for 2 out
// of 3. Zero when there were no operations.
func (r Result) WinRate() float64 {
	if r.TotalOperations == 0 {
		return 0
	}
	return float64(r.Gains) / float64(r.TotalOperations) * 100
}

// MaxDrawdownPercentage is the largest peak-to-trough decline in balance
// across the operations that produced this Result, as a percentage of the
// peak (e.g. 15.23 for a 15.23% drawdown). Zero if the balance never
// declined below a prior peak (including when there were no operations).
func (r Result) MaxDrawdownPercentage() float64 {
	return r.maxDrawdownPercentage
}

// MaxDrawdownAmount is the largest peak-to-trough decline in balance across
// the operations that produced this Result, in absolute currency terms
// (always non-negative). Zero under the same conditions as
// MaxDrawdownPercentage.
func (r Result) MaxDrawdownAmount() money.Money {
	return r.maxDrawdownAmount
}

// resultJSON mirrors Result but with money.Money fields converted to plain
// JSON numbers (see domain.Candle's candleJSON for the same pattern), and
// Operations marked omitempty so API callers can drop it (e.g. when the
// caller didn't ask for verbose output) by nil-ing it out before marshaling.
type resultJSON struct {
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

func (r Result) MarshalJSON() ([]byte, error) {
	maxDrawdownAmount := r.MaxDrawdownAmount()
	return json.Marshal(resultJSON{
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
