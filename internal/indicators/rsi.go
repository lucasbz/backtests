package indicators

import "github.com/lucasbz/backtests/internal/domain"

// RSI is a Wilder's-smoothing relative strength index over the last Period
// candle-to-candle price changes. It needs Period+1 candles (Period
// deltas) before it's ready: the first Period deltas seed the average
// gain/average loss as a plain arithmetic mean (matching Wilder's original
// method), and every delta after that blends in via Wilder's smoothing:
//
//	avgGain = (prevAvgGain*(Period-1) + gain) / Period
//	avgLoss = (prevAvgLoss*(Period-1) + loss) / Period
//	rs      = avgGain / avgLoss
//	rsi     = 100 - 100/(1+rs)
//
// where a given delta contributes to gain (and 0 to loss) when the close
// rose, or to loss (and 0 to gain) when it fell; an unchanged close
// contributes 0 to both.
type RSI struct {
	Period int

	prevClose  float64
	hasPrev    bool
	deltaCount int
	avgGain    float64
	avgLoss    float64
	ready      bool
}

func (r *RSI) Update(candle domain.Candle) {
	price := candle.Close.AsMajorUnits()

	if !r.hasPrev {
		r.prevClose = price
		r.hasPrev = true
		return
	}

	delta := price - r.prevClose
	r.prevClose = price

	gain, loss := 0.0, 0.0
	switch {
	case delta > 0:
		gain = delta
	case delta < 0:
		loss = -delta
	}

	if r.Period <= 0 {
		return
	}

	if r.deltaCount < r.Period {
		// Warm-up: accumulate a plain sum of the first Period deltas'
		// gains/losses, divided down to an average once the Period-th one
		// arrives.
		r.avgGain += gain
		r.avgLoss += loss
		r.deltaCount++
		if r.deltaCount == r.Period {
			r.avgGain /= float64(r.Period)
			r.avgLoss /= float64(r.Period)
			r.ready = true
		}
		return
	}

	r.avgGain = (r.avgGain*float64(r.Period-1) + gain) / float64(r.Period)
	r.avgLoss = (r.avgLoss*float64(r.Period-1) + loss) / float64(r.Period)
}

func (r *RSI) Value() (float64, bool) {
	if !r.ready {
		return 0, false
	}

	if r.avgLoss == 0 {
		if r.avgGain == 0 {
			// No movement at all over the window - genuinely neutral,
			// distinct from "gains but never a loss" (100).
			return 50, true
		}
		return 100, true
	}

	rs := r.avgGain / r.avgLoss
	return 100 - (100 / (1 + rs)), true
}
