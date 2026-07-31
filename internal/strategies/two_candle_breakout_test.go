package strategies

import (
	"testing"

	"github.com/lucasbz/backtests/internal/domain"
)

func TestTwoCandleBreakout_TooFewCandles(t *testing.T) {
	s := &TwoCandleBreakout{}
	candles := []domain.Candle{
		candleWithLowHigh("2010-01-04", 1200, 1242),
		candleWithLowHigh("2010-01-05", 1200, 1242),
	}

	if ops := runStrategy(s, newMoney(10000), candles); ops != nil {
		t.Errorf("ops = %+v, want nil (fewer than 3 candles)", ops)
	}
}

func TestTwoCandleBreakout_DropsUnclosedPositionAtEndOfCandles(t *testing.T) {
	s := &TwoCandleBreakout{}
	// window(c0,c1) low-min = min(1200,1100) = 1100; c2's low (1050) touches
	// it -> buy @1100. Candles end right there, with no candle ever reaching
	// the sell trigger, so the position is left open and must not be
	// reported as a completed Operation.
	candles := []domain.Candle{
		candleWithLowHigh("2010-01-04", 1200, 1300),
		candleWithLowHigh("2010-01-05", 1100, 1250),
		candleWithLowHigh("2010-01-06", 1050, 1150),
	}

	if ops := runStrategy(s, newMoney(10000), candles); ops != nil {
		t.Fatalf("ops = %+v, want nil (open position at end of candles must be dropped)", ops)
	}
}

func TestTwoCandleBreakout_BuysAndSellsOnTouch(t *testing.T) {
	s := &TwoCandleBreakout{}
	// last two candles before the 3rd: lows 1200/1100 -> min 1100. 3rd
	// candle's low reaches 1100, triggering a buy at 1100.
	//
	// now holding: last two candles are 01-05 (high 1250) and 01-06 (high
	// 1150) -> max 1250. The 4th candle's high reaches 1250, selling.
	candles := []domain.Candle{
		candleWithLowHigh("2010-01-04", 1200, 1300),
		candleWithLowHigh("2010-01-05", 1100, 1250),
		candleWithLowHigh("2010-01-06", 1050, 1150),
		candleWithLowHigh("2010-01-07", 1200, 1250),
	}

	ops := runStrategy(s, newMoney(10000), candles)
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}

	op := ops[0]
	assertOrder(t, "BuyOrder", op.BuyOrder, "2010-01-06", 1100, 9, domain.Buy)
	assertOrder(t, "SellOrder", op.SellOrder, "2010-01-07", 1250, 9, domain.Sell)
}

func TestTwoCandleBreakout_WaitsUntilPriceTouchesTrigger(t *testing.T) {
	// min of last two lows is 1100, but the 3rd candle's low (1150) never
	// reaches it, so no buy yet.
	threeCandles := []domain.Candle{
		candleWithLowHigh("2010-01-04", 1200, 1300),
		candleWithLowHigh("2010-01-05", 1100, 1250),
		candleWithLowHigh("2010-01-06", 1150, 1220),
	}

	// Runs against its own strategy instance (rather than reusing the one
	// above) so this call's Decide history doesn't leak into the one below
	// - runStrategy replays candles from scratch each call, but a
	// TwoCandleBreakout instance's internal window persists across calls.
	if ops := runStrategy(&TwoCandleBreakout{}, newMoney(10000), threeCandles); ops != nil {
		t.Fatalf("ops = %+v, want nil (trigger price never touched)", ops)
	}

	// next window: last two lows (1100, 1150) -> min 1100. This candle's low
	// (1090) reaches it, triggering the buy - but candles end right there
	// (still holding, no sell yet), so no completed operation is reported.
	fourCandles := append(threeCandles, candleWithLowHigh("2010-01-07", 1090, 1200))

	if ops := runStrategy(&TwoCandleBreakout{}, newMoney(10000), fourCandles); len(ops) != 0 {
		t.Fatalf("got %d completed operations, want 0 (still holding, no sell yet): %+v", len(ops), ops)
	}
}

func TestTwoCandleBreakout_ReinvestsProceedsIntoNextCycle(t *testing.T) {
	s := &TwoCandleBreakout{}

	// cycle 1: window(c0,c1) low-min = min(200,150) = 150; c2's low (100)
	// touches it -> buy @150, qty = 1000/150 = 6.
	// window(c1,c2) high-max = max(250,200) = 250; c3's high (400) touches
	// it -> sell @250, qty 6 -> proceeds = 250*6 = 1500.
	// cycle 2: window(c2,c3) low-min = min(100,300) = 100; c4's low (90)
	// touches it -> buy @100. Reinvesting the 1500 proceeds (not the
	// original 1000 balance) gives qty = 1500/100 = 15.
	// window(c3,c4) high-max = max(400,500) = 500; c5's high (550) touches
	// it -> sell @500, qty 15.
	candles := []domain.Candle{
		candleWithLowHigh("2010-01-04", 200, 300),
		candleWithLowHigh("2010-01-05", 150, 250),
		candleWithLowHigh("2010-01-06", 100, 200),
		candleWithLowHigh("2010-01-07", 300, 400),
		candleWithLowHigh("2010-01-08", 90, 500),
		candleWithLowHigh("2010-01-09", 400, 550),
	}

	ops := runStrategy(s, newMoney(1000), candles)
	if len(ops) != 2 {
		t.Fatalf("got %d operations, want 2: %+v", len(ops), ops)
	}

	assertOrder(t, "cycle 1 BuyOrder", ops[0].BuyOrder, "2010-01-06", 150, 6, domain.Buy)
	assertOrder(t, "cycle 1 SellOrder", ops[0].SellOrder, "2010-01-07", 250, 6, domain.Sell)

	// The key assertion: qty 15 only arises from reinvesting the 1500
	// proceeds of cycle 1. The original 1000 balance would have bought only
	// 10 shares at 100.
	assertOrder(t, "cycle 2 BuyOrder", ops[1].BuyOrder, "2010-01-08", 100, 15, domain.Buy)
	assertOrder(t, "cycle 2 SellOrder", ops[1].SellOrder, "2010-01-09", 500, 15, domain.Sell)
}

func TestTwoCandleBreakout_InsufficientBalanceSkipsBuy(t *testing.T) {
	s := &TwoCandleBreakout{}
	candles := []domain.Candle{
		candleWithLowHigh("2010-01-04", 1200, 1300),
		candleWithLowHigh("2010-01-05", 1100, 1250),
		candleWithLowHigh("2010-01-06", 1050, 1150),
	}

	if ops := runStrategy(s, newMoney(500), candles); ops != nil {
		t.Errorf("ops = %+v, want nil (balance can't afford one share)", ops)
	}
}
