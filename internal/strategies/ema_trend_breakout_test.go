package strategies

import (
	"strings"
	"testing"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

// candleWithLowHighClose builds a domain.Candle with Low, High and Close all
// set - EMATrendBreakout needs Low/High for its inherited two-candle window
// trigger (see TwoCandleBreakout) and Close for its EMA trend filter, unlike
// candleWithLowHigh (window-only strategies) or candleWithClose
// (indicator-only strategies) in buy_and_hold_test.go, neither of which sets
// all three.
func candleWithLowHighClose(date string, low, high, close int64) domain.Candle {
	return domain.Candle{Date: date, Low: newMoney(low), High: newMoney(high), Close: newMoney(close)}
}

func TestEMATrendBreakout_Name(t *testing.T) {
	s := &EMATrendBreakout{}
	if got := s.Name(); got != "EMA Trend Breakout" {
		t.Errorf("Name() = %q, want %q", got, "EMA Trend Breakout")
	}
}

// TestEMATrendBreakout_WarmUp checks that no buy fires before both EMAs are
// ready, even though the two-candle window trigger fires right on schedule.
// shortPeriod=2 is ready starting candle index 1; longPeriod=4 needs 4
// candles, so it's still warming up at index 2, exactly when the window
// trigger (min(c0.Low, c1.Low)=1100, reached by c2.Low=1050) would otherwise
// buy.
func TestEMATrendBreakout_WarmUp(t *testing.T) {
	s, err := newEMATrendBreakout(map[string]float64{"shortPeriod": 2, "longPeriod": 4})
	if err != nil {
		t.Fatalf("newEMATrendBreakout: %v", err)
	}

	candles := []domain.Candle{
		candleWithLowHighClose("2010-01-04", 1200, 1300, 1000),
		candleWithLowHighClose("2010-01-05", 1100, 1250, 1000),
		candleWithLowHighClose("2010-01-06", 1050, 1150, 1000),
	}

	if ops := runStrategy(s, newMoney(10000), candles); ops != nil {
		t.Errorf("ops = %+v, want nil (long EMA not ready yet)", ops)
	}
}

// TestEMATrendBreakout_WindowTriggerButDowntrendBlocksBuy uses
// shortPeriod=2/longPeriod=3 with closes 1000, 1000... wait see below: at
// c2 the window trigger fires (min(c0.Low=1200, c1.Low=1100)=1100, c2.Low
// 1050 reaches it), and both EMAs are ready (short since c1, long seeded at
// c2), but the trend is down: short EMA(2) blends closes 4000,4000,1000 to
// (1000-4000)*(2/3)+4000=2000, long EMA(3) seeds at avg(4000,4000,1000)
// =3000. short (2000) < long (3000), so the uptrend condition fails and no
// buy is placed despite the window trigger.
func TestEMATrendBreakout_WindowTriggerButDowntrendBlocksBuy(t *testing.T) {
	s, err := newEMATrendBreakout(map[string]float64{"shortPeriod": 2, "longPeriod": 3})
	if err != nil {
		t.Fatalf("newEMATrendBreakout: %v", err)
	}

	candles := []domain.Candle{
		candleWithLowHighClose("2010-01-04", 1200, 1300, 4000),
		candleWithLowHighClose("2010-01-05", 1100, 1250, 4000),
		candleWithLowHighClose("2010-01-06", 1050, 1150, 1000),
	}

	if ops := runStrategy(s, newMoney(10000), candles); ops != nil {
		t.Errorf("ops = %+v, want nil (short EMA below long EMA, buy blocked)", ops)
	}
}

// TestEMATrendBreakout_WindowTriggerAndUptrendBuys uses the same
// shortPeriod=2/longPeriod=3 shape as the downtrend test above, but with
// rising closes 1000, 1000, 4000: short EMA(2) seeds at avg(1000,1000)=1000
// then blends to (4000-1000)*(2/3)+1000=3000; long EMA(3) seeds at
// avg(1000,1000,4000)=2000. short (3000) > long (2000) confirms the
// uptrend, and c2's close (4000) is above the short EMA (3000) too, so the
// window trigger (min(1200,1100)=1100, reached by c2.Low=1050) fires a buy
// at 1100 - the window's minPrice, not the EMA value or the close.
func TestEMATrendBreakout_WindowTriggerAndUptrendBuys(t *testing.T) {
	s, err := newEMATrendBreakout(map[string]float64{"shortPeriod": 2, "longPeriod": 3})
	if err != nil {
		t.Fatalf("newEMATrendBreakout: %v", err)
	}

	candles := []domain.Candle{
		candleWithLowHighClose("2010-01-04", 1200, 1300, 1000),
		candleWithLowHighClose("2010-01-05", 1100, 1250, 1000),
		candleWithLowHighClose("2010-01-06", 1050, 1150, 4000),
	}

	position := runToOpenPosition(t, s, newMoney(10000), candles)
	assertOrder(t, "BuyOrder", position.Buy, "2010-01-06", 1100, 9, domain.Buy)
}

// TestEMATrendBreakout_SellIgnoresEMA proves the sell/exit branch is
// completely unaffected by EMA state, unlike the buy branch: it constructs
// an EMATrendBreakout with longPeriod=100 (so the long EMA has had only one
// Update call by the time Decide runs below and is nowhere near ready),
// manually seeds its two-candle window and hands it an already-open
// position directly (bypassing the buy path entirely, since a real buy
// couldn't happen with the long EMA unready). If the sell branch checked
// EMA readiness or trend the way the buy branch does, this would return
// nil; instead it must sell purely because the window's high trigger
// (max(1250, 1150)=1250) is reached by the candle's high (1300).
func TestEMATrendBreakout_SellIgnoresEMA(t *testing.T) {
	strategy, err := newEMATrendBreakout(map[string]float64{"shortPeriod": 1, "longPeriod": 100})
	if err != nil {
		t.Fatalf("newEMATrendBreakout: %v", err)
	}
	s := strategy.(*EMATrendBreakout)
	s.window = []domain.Candle{
		candleWithLowHighClose("2010-01-05", 1100, 1250, 1000),
		candleWithLowHighClose("2010-01-06", 1050, 1150, 1000),
	}

	position := &domain.Position{Buy: domain.Order{Price: newMoney(1100), Quantity: 9, OrderType: domain.Buy}}
	candle := candleWithLowHighClose("2010-01-07", 1200, 1300, 1000)

	order := s.Decide(candle, position, false)
	if order == nil {
		t.Fatal("Decide = nil, want a sell order (sell must not be gated by EMA readiness/trend)")
	}
	if order.OrderType != domain.Sell {
		t.Errorf("OrderType = %q, want %q", order.OrderType, domain.Sell)
	}
	if order.Price.Amount() != 1250 {
		t.Errorf("Price = %d, want %d", order.Price.Amount(), 1250)
	}
	if order.Quantity != 9 {
		t.Errorf("Quantity = %d, want %d", order.Quantity, 9)
	}
}

func TestNewEMATrendBreakout_MissingShortPeriod(t *testing.T) {
	_, err := newEMATrendBreakout(map[string]float64{"longPeriod": 80})
	if err == nil {
		t.Fatal("expected error for missing shortPeriod")
	}
	if !strings.Contains(err.Error(), "shortPeriod") {
		t.Errorf("error = %q, want it to mention shortPeriod", err.Error())
	}
}

func TestNewEMATrendBreakout_MissingLongPeriod(t *testing.T) {
	_, err := newEMATrendBreakout(map[string]float64{"shortPeriod": 8})
	if err == nil {
		t.Fatal("expected error for missing longPeriod")
	}
	if !strings.Contains(err.Error(), "longPeriod") {
		t.Errorf("error = %q, want it to mention longPeriod", err.Error())
	}
}

func TestNewEMATrendBreakout_ShortNotLessThanLong(t *testing.T) {
	cases := []map[string]float64{
		{"shortPeriod": 80, "longPeriod": 80},
		{"shortPeriod": 100, "longPeriod": 80},
	}
	for _, params := range cases {
		if _, err := newEMATrendBreakout(params); err == nil {
			t.Errorf("newEMATrendBreakout(%+v) = nil error, want an error (shortPeriod must be < longPeriod)", params)
		}
	}
}

func TestLoadStrategy_EMATrendBreakout(t *testing.T) {
	s, err := LoadStrategy("ema-trend-breakout", map[string]float64{"shortPeriod": 8, "longPeriod": 80})
	if err != nil {
		t.Fatalf("LoadStrategy: %v", err)
	}
	if got := s.Name(); got != "EMA Trend Breakout" {
		t.Errorf("Name() = %q, want %q", got, "EMA Trend Breakout")
	}
}

// runToOpenPosition drives candles through strategy exactly like
// runStrategy, but returns the still-open domain.Position instead of
// requiring the backtest to also close it - useful for tests (like the
// uptrend-buy test above) that only need to assert on the buy leg and don't
// want to also have to engineer a matching sell candle.
func runToOpenPosition(t *testing.T, strategy domain.Strategy, balance money.Money, candles []domain.Candle) *domain.Position {
	t.Helper()
	var position *domain.Position

	for i, candle := range candles {
		isLast := i == len(candles)-1
		if position != nil {
			t.Fatalf("candle %d: position already open, runToOpenPosition only supports a single buy leg", i)
		}
		order := strategy.Decide(candle, nil, isLast)
		if order == nil {
			continue
		}
		order.Quantity = balance.Amount() / order.Price.Amount()
		if order.Quantity <= 0 {
			t.Fatalf("candle %d: order quantity <= 0", i)
		}
		position = &domain.Position{Buy: *order}
	}

	if position == nil {
		t.Fatal("runToOpenPosition: no buy order was ever placed")
	}
	return position
}
