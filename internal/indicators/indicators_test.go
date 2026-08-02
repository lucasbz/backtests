package indicators

import (
	"math"
	"testing"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

// closeCandle builds a domain.Candle whose only field this package's
// indicators read is Close, set from close (a BRL major-units value, e.g.
// 19.9 for R$19.90).
func closeCandle(close float64) domain.Candle {
	return domain.Candle{Close: *money.NewFromFloat(close, domain.Currency)}
}

func feed(ind Indicator, closes ...float64) {
	for _, c := range closes {
		ind.Update(closeCandle(c))
	}
}

func assertNotReady(t *testing.T, ind Indicator, label string) {
	t.Helper()
	if _, ready := ind.Value(); ready {
		t.Errorf("%s: Value() ready = true, want false (still warming up)", label)
	}
}

func assertValue(t *testing.T, ind Indicator, label string, want float64) {
	t.Helper()
	got, ready := ind.Value()
	if !ready {
		t.Fatalf("%s: Value() ready = false, want true", label)
	}
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("%s: Value() = %v, want %v", label, got, want)
	}
}

// --- SMA ---

func TestSMA_WarmUp(t *testing.T) {
	sma := &SMA{Period: 3}
	assertNotReady(t, sma, "before any candle")

	feed(sma, 10)
	assertNotReady(t, sma, "after 1/3 candles")

	feed(sma, 20)
	assertNotReady(t, sma, "after 2/3 candles")
}

// TestSMA_HandComputed feeds 10, 20, 30, 40 through a period-3 SMA and
// checks it against a hand-computed average at each ready point:
// avg(10,20,30) = 20, then the window slides to avg(20,30,40) = 30.
func TestSMA_HandComputed(t *testing.T) {
	sma := &SMA{Period: 3}
	feed(sma, 10, 20, 30)
	assertValue(t, sma, "after 3 candles", 20)

	feed(sma, 40)
	assertValue(t, sma, "after 4 candles (window slides)", 30)
}

func TestSMA_InvalidPeriodNeverReady(t *testing.T) {
	sma := &SMA{Period: 0}
	feed(sma, 10, 20, 30)
	assertNotReady(t, sma, "period 0")
}

// --- EMA ---

func TestEMA_WarmUp(t *testing.T) {
	ema := &EMA{Period: 2}
	assertNotReady(t, ema, "before any candle")

	feed(ema, 10)
	assertNotReady(t, ema, "after 1/2 candles")
}

// TestEMA_HandComputed feeds 10, 12, 11, 13, 14 through a period-2 EMA.
// Seed (the SMA of the first 2 closes) = avg(10,12) = 11, ready at that
// point. Each later close blends in via multiplier 2/(2+1) = 2/3:
//
//	ema(11) = (11-11)*(2/3)+11        = 11
//	ema(13) = (13-11)*(2/3)+11        = 12.333333...
//	ema(14) = (14-12.333333)*(2/3)+12.333333 = 13.444444...
func TestEMA_HandComputed(t *testing.T) {
	ema := &EMA{Period: 2}
	feed(ema, 10, 12)
	assertValue(t, ema, "seeded after 2 candles", 11)

	feed(ema, 11)
	assertValue(t, ema, "3rd candle", 11)

	feed(ema, 13)
	assertValue(t, ema, "4th candle", 12.333333)

	feed(ema, 14)
	assertValue(t, ema, "5th candle", 13.444444)
}

func TestEMA_InvalidPeriodNeverReady(t *testing.T) {
	ema := &EMA{Period: 0}
	feed(ema, 10, 20, 30)
	assertNotReady(t, ema, "period 0")
}

// --- RSI ---

func TestRSI_WarmUp(t *testing.T) {
	rsi := &RSI{Period: 2}
	assertNotReady(t, rsi, "before any candle")

	feed(rsi, 10) // only sets prevClose, no delta yet
	assertNotReady(t, rsi, "after 1 candle (no delta yet)")

	feed(rsi, 12) // 1st delta
	assertNotReady(t, rsi, "after 1/2 deltas")
}

// TestRSI_HandComputed feeds closes 10, 12, 11, 13, 14 through a period-2
// RSI (deltas: +2, -1, +2, +1):
//
//	seed (first 2 deltas): avgGain=(2+0)/2=1, avgLoss=(0+1)/2=0.5
//	  -> rsi = 100 - 100/(1+1/0.5) = 66.666...
//	3rd delta (+2): avgGain=(1*1+2)/2=1.5, avgLoss=(0.5*1+0)/2=0.25
//	  -> rsi = 100 - 100/(1+1.5/0.25) = 85.714285...
//	4th delta (+1): avgGain=(1.5*1+1)/2=1.25, avgLoss=(0.25*1+0)/2=0.125
//	  -> rsi = 100 - 100/(1+1.25/0.125) = 90.909090...
func TestRSI_HandComputed(t *testing.T) {
	rsi := &RSI{Period: 2}
	feed(rsi, 10, 12, 11)
	assertValue(t, rsi, "after seed (2 deltas)", 66.666667)

	feed(rsi, 13)
	assertValue(t, rsi, "after 3rd delta", 85.714286)

	feed(rsi, 14)
	assertValue(t, rsi, "after 4th delta", 90.909091)
}

// TestRSI_AllGainsIsHundred checks the avgLoss==0 special case: a close
// that only ever rises produces RSI 100, not a divide-by-zero.
func TestRSI_AllGainsIsHundred(t *testing.T) {
	rsi := &RSI{Period: 2}
	feed(rsi, 10, 11, 12)
	assertValue(t, rsi, "all gains", 100)
}

// TestRSI_AllLossesIsZero checks the avgGain==0 case: a close that only
// ever falls produces RSI 0.
func TestRSI_AllLossesIsZero(t *testing.T) {
	rsi := &RSI{Period: 2}
	feed(rsi, 12, 11, 10)
	assertValue(t, rsi, "all losses", 0)
}

// TestRSI_NoMovementIsFifty checks the avgGain==0 && avgLoss==0 case: a
// perfectly flat close is treated as neutral (50), not "all losses" (0).
func TestRSI_NoMovementIsFifty(t *testing.T) {
	rsi := &RSI{Period: 2}
	feed(rsi, 10, 10, 10)
	assertValue(t, rsi, "no movement", 50)
}

func TestRSI_InvalidPeriodNeverReady(t *testing.T) {
	rsi := &RSI{Period: 0}
	feed(rsi, 10, 12, 11, 13)
	assertNotReady(t, rsi, "period 0")
}

// --- registry ---

func TestAvailableIndicatorNamesList_Sorted(t *testing.T) {
	names := AvailableIndicatorNamesList()
	want := []string{"ema", "rsi", "sma"}
	if len(names) != len(want) {
		t.Fatalf("AvailableIndicatorNamesList() = %v, want %v", names, want)
	}
	for i, name := range want {
		if names[i] != name {
			t.Errorf("AvailableIndicatorNamesList()[%d] = %q, want %q", i, names[i], name)
		}
	}
}

func TestAvailableIndicatorNames(t *testing.T) {
	got := AvailableIndicatorNames()
	if got != "ema, rsi, sma" {
		t.Errorf("AvailableIndicatorNames() = %q, want %q", got, "ema, rsi, sma")
	}
}

func TestLoadIndicator_SMA(t *testing.T) {
	ind, err := LoadIndicator("sma", map[string]float64{"period": 5})
	if err != nil {
		t.Fatalf("LoadIndicator: %v", err)
	}
	sma, ok := ind.(*SMA)
	if !ok {
		t.Fatalf("LoadIndicator(%q) = %T, want *SMA", "sma", ind)
	}
	if sma.Period != 5 {
		t.Errorf("Period = %d, want 5", sma.Period)
	}
}

func TestLoadIndicator_EMA(t *testing.T) {
	ind, err := LoadIndicator("ema", map[string]float64{"period": 12})
	if err != nil {
		t.Fatalf("LoadIndicator: %v", err)
	}
	ema, ok := ind.(*EMA)
	if !ok {
		t.Fatalf("LoadIndicator(%q) = %T, want *EMA", "ema", ind)
	}
	if ema.Period != 12 {
		t.Errorf("Period = %d, want 12", ema.Period)
	}
}

func TestLoadIndicator_RSI(t *testing.T) {
	ind, err := LoadIndicator("rsi", map[string]float64{"period": 14})
	if err != nil {
		t.Fatalf("LoadIndicator: %v", err)
	}
	rsi, ok := ind.(*RSI)
	if !ok {
		t.Fatalf("LoadIndicator(%q) = %T, want *RSI", "rsi", ind)
	}
	if rsi.Period != 14 {
		t.Errorf("Period = %d, want 14", rsi.Period)
	}
}

func TestLoadIndicator_Unknown(t *testing.T) {
	ind, err := LoadIndicator("does-not-exist", map[string]float64{"period": 5})
	if err == nil {
		t.Fatal("LoadIndicator(unknown) = nil error, want an error")
	}
	if ind != nil {
		t.Errorf("LoadIndicator(unknown) = %+v, want nil", ind)
	}
}

func TestLoadIndicator_MissingPeriod(t *testing.T) {
	for _, name := range []string{"sma", "ema", "rsi"} {
		if _, err := LoadIndicator(name, map[string]float64{}); err == nil {
			t.Errorf("LoadIndicator(%q, {}) = nil error, want an error for missing period", name)
		}
	}
}

func TestLoadIndicator_InvalidPeriod(t *testing.T) {
	cases := []float64{0, -1, 1.5}
	for _, name := range []string{"sma", "ema", "rsi"} {
		for _, period := range cases {
			if _, err := LoadIndicator(name, map[string]float64{"period": period}); err == nil {
				t.Errorf("LoadIndicator(%q, {period: %v}) = nil error, want an error", name, period)
			}
		}
	}
}
