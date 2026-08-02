package indicators

import "github.com/lucasbz/backtests/internal/domain"

// EMA is a standard exponential moving average: after an initial seed
// (the simple average of the first Period candles' closes, same as SMA
// would produce over that window), each subsequent close is blended in
// via the standard multiplier 2/(Period+1):
//
//	ema = (close - prevEMA) * multiplier + prevEMA
//
// It's ready once it has seen Period candles (the seed).
type EMA struct {
	Period int

	// seedWindow accumulates closes until Period of them have been seen,
	// at which point their average seeds value and seedWindow is dropped -
	// mirrors SMA's window, but only needed during warm-up.
	seedWindow []float64
	value      float64
	seeded     bool
}

func (e *EMA) Update(candle domain.Candle) {
	price := candle.Close.AsMajorUnits()

	if e.seeded {
		multiplier := 2 / float64(e.Period+1)
		e.value = (price-e.value)*multiplier + e.value
		return
	}

	e.seedWindow = append(e.seedWindow, price)
	if e.Period <= 0 || len(e.seedWindow) < e.Period {
		return
	}

	sum := 0.0
	for _, v := range e.seedWindow {
		sum += v
	}
	e.value = sum / float64(e.Period)
	e.seeded = true
	e.seedWindow = nil
}

func (e *EMA) Value() (float64, bool) {
	if e.Period <= 0 {
		return 0, false
	}
	return e.value, e.seeded
}
