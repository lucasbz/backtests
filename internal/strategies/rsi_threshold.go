package strategies

import (
	"fmt"

	"github.com/lucasbz/backtests/internal/domain"
	"github.com/lucasbz/backtests/internal/indicators"
)

type RSIThreshold struct {
	rsi                indicators.Indicator
	oversold           float64
	overbought         float64
	wasBelowOversold   *bool
	wasAboveOverbought *bool
}

func newRSIThreshold(params map[string]float64) (domain.Strategy, error) {
	period, ok := params["period"]
	if !ok {
		return nil, fmt.Errorf("rsi-threshold: missing required %q param", "period")
	}
	oversold, ok := params["oversold"]
	if !ok {
		return nil, fmt.Errorf("rsi-threshold: missing required %q param", "oversold")
	}
	overbought, ok := params["overbought"]
	if !ok {
		return nil, fmt.Errorf("rsi-threshold: missing required %q param", "overbought")
	}
	if oversold < 0 || oversold > 100 {
		return nil, fmt.Errorf("rsi-threshold: oversold must be between 0 and 100, got %v", oversold)
	}
	if overbought < 0 || overbought > 100 {
		return nil, fmt.Errorf("rsi-threshold: overbought must be between 0 and 100, got %v", overbought)
	}
	if oversold >= overbought {
		return nil, fmt.Errorf("rsi-threshold: oversold (%v) must be less than overbought (%v)", oversold, overbought)
	}

	rsi, err := indicators.LoadIndicator("rsi", map[string]float64{"period": period})
	if err != nil {
		return nil, fmt.Errorf("rsi-threshold: %w", err)
	}

	return &RSIThreshold{rsi: rsi, oversold: oversold, overbought: overbought}, nil
}

func (s *RSIThreshold) Name() string {
	return "RSI Threshold"
}

// Decide implements domain.Strategy. It ignores isLast, same as
// TwoCandleBreakout: an unclosed position at the end of the backtest is
// left open and dropped, not force-sold. See crossoverStrategy.Decide's
// doc comment for why RSI is updated with the current candle BEFORE
// Value() is read: the RSI value read here is the entire signal (not
// paired with a separate check against today's raw data, the way
// TwoCandleBreakout's window is), so it must reflect the same candle
// whose Close is used as the fill price below.
func (s *RSIThreshold) Decide(candle domain.Candle, position *domain.Position, isLast bool) *domain.Order {
	s.rsi.Update(candle)

	value, ready := s.rsi.Value()
	if !ready {
		return nil // still warming up
	}

	belowOversold := value < s.oversold
	aboveOverbought := value > s.overbought
	defer func() {
		s.wasBelowOversold = &belowOversold
		s.wasAboveOverbought = &aboveOverbought
	}()
	if s.wasBelowOversold == nil {
		return nil // first ready candle - no prior state to compare against, no cross yet
	}

	crossedBelowOversold := !*s.wasBelowOversold && belowOversold
	crossedAboveOverbought := !*s.wasAboveOverbought && aboveOverbought

	if crossedBelowOversold && position == nil {
		return &domain.Order{
			Date:      candle.Date,
			Price:     candle.Close,
			OrderType: domain.Buy,
		}
	}
	if crossedAboveOverbought && position != nil {
		return &domain.Order{
			Date:      candle.Date,
			Price:     candle.Close,
			Quantity:  position.Buy.Quantity,
			OrderType: domain.Sell,
		}
	}
	return nil
}
