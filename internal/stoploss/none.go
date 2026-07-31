package stoploss

import "github.com/lucasbz/backtests/internal/domain"

// NoStopLoss is an explicit no-op domain.StopLoss: Check never triggers, for
// any candle/position combination. It exists so callers can select "no
// stop-loss" as a real, explicit type (e.g. "none" via LoadStopLoss or the
// API), rather than only being able to express that by omitting a stop-loss
// configuration entirely.
type NoStopLoss struct{}

func (s *NoStopLoss) Name() string {
	return "None"
}

// Check implements domain.StopLoss. It never triggers.
func (s *NoStopLoss) Check(candle domain.Candle, position domain.Position) *domain.Order {
	return nil
}
