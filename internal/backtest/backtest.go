package backtest

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/cotahist"
	"github.com/lucasbz/backtests/internal/domain"
)

// Backtest is the configuration for a single backtest run: which ticker,
// over what timeframe, starting from how much cash, using which strategy.
type Backtest struct {
	Ticker   string
	Start    time.Time
	End      time.Time
	Balance  money.Money
	Strategy domain.Strategy
}

// Result summarizes the outcome of a Backtest.Run: every operation the
// strategy produced, and how the balance moved because of them.
type Result struct {
	StrategyName    string
	Operations      []domain.Operation
	StartingBalance money.Money
	EndingBalance   money.Money
	Profit          money.Money
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
		Gains:            r.Gains,
		Losses:           r.Losses,
		WinRate:          r.WinRate,
		Operations:       r.Operations,
	})
}

// Run feeds b.Strategy every candle for b.Ticker within [b.Start, b.End], in
// date order, so the strategy can decide whether to buy or sell on each one,
// then compiles the resulting operations into a Result.
func (b *Backtest) Run() (*Result, error) {
	candles, err := cotahist.LoadCandles(b.Ticker, b.Start, b.End)
	if err != nil {
		return nil, fmt.Errorf("loading candles for %s: %w", b.Ticker, err)
	}

	for _, candle := range candles {
		b.Strategy.Traverse(candle)
	}

	return compileResult(b.Strategy, b.Balance)
}

// compileResult turns a strategy's operations into a Result, given the
// backtest's starting balance: go over each operation, adding its gain or
// loss to the running total profit and balance, and counting it as a gain
// or a loss.
func compileResult(strategy domain.Strategy, startingBalance money.Money) (*Result, error) {
	operations := strategy.Operations()

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
	if total := len(operations); total > 0 {
		winRate = float64(gains) / float64(total) * 100
	}

	return &Result{
		StrategyName:     strategy.Name(),
		Operations:       operations,
		StartingBalance:  startingBalance,
		EndingBalance:    balance,
		Profit:           *profit,
		Gains:            gains,
		Losses:           losses,
		ProfitPercentage: profitPercentage,
		WinRate:          winRate,
	}, nil
}
