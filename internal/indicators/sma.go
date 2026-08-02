package indicators

import "github.com/lucasbz/backtests/internal/domain"

// SMA is a simple moving average over the last Period candles' closes:
// their plain arithmetic mean. It's ready once it has seen Period candles.
type SMA struct {
	Period int

	// window holds the closing prices of at most the last Period candles
	// seen, oldest first. sum is kept in sync with window's contents so
	// Value doesn't need to re-sum on every call.
	window []float64
	sum    float64
}

func (s *SMA) Update(candle domain.Candle) {
	price := candle.Close.AsMajorUnits()

	s.window = append(s.window, price)
	s.sum += price
	if len(s.window) > s.Period {
		s.sum -= s.window[0]
		s.window = s.window[1:]
	}
}

func (s *SMA) Value() (float64, bool) {
	if s.Period <= 0 || len(s.window) < s.Period {
		return 0, false
	}
	return s.sum / float64(s.Period), true
}
