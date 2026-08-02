package strategies

import (
	"testing"

	"github.com/Rhymond/go-money"
	"github.com/lucasbz/backtests/internal/domain"
)

func newMoney(amount int64) money.Money {
	return *money.New(amount, domain.Currency)
}

func candleWithLowHigh(date string, low, high int64) domain.Candle {
	return domain.Candle{Date: date, Low: newMoney(low), High: newMoney(high)}
}

// candleWithClose builds a domain.Candle with only Close set - the field
// the indicator-backed strategies (crossoverStrategy, RSIThreshold) read,
// mirroring candleWithLowHigh's convention above. close is in cents (same
// units as newMoney), e.g. 2000 for R$20.00.
func candleWithClose(date string, close int64) domain.Candle {
	return domain.Candle{Date: date, Close: newMoney(close)}
}

func assertOrder(t *testing.T, label string, got domain.Order, wantDate string, wantPrice int64, wantQuantity int64, wantType domain.OrderType) {
	t.Helper()
	if got.Date != wantDate {
		t.Errorf("%s.Date = %q, want %q", label, got.Date, wantDate)
	}
	if got.Price.Amount() != wantPrice {
		t.Errorf("%s.Price = %d, want %d", label, got.Price.Amount(), wantPrice)
	}
	if got.Quantity != wantQuantity {
		t.Errorf("%s.Quantity = %d, want %d", label, got.Quantity, wantQuantity)
	}
	if got.OrderType != wantType {
		t.Errorf("%s.OrderType = %q, want %q", label, got.OrderType, wantType)
	}
}

// runStrategy drives candles one at a time through strategy exactly the way
// internal/backtest.Backtest.Run does (see traverse in
// internal/backtest/backtest.go): balance tracking, quantity sizing, and
// Position lifecycle all live here in the test now, mirroring what moved
// out of each strategy and into Backtest. Strategy tests use this instead
// of calling Decide by hand so they keep exercising the same integration
// behavior (quantity sizing, reinvestment, dropping unclosed positions)
// they did before the domain.Strategy interface changed, without
// depending on the backtest package itself.
func runStrategy(strategy domain.Strategy, balance money.Money, candles []domain.Candle) []domain.Operation {
	var position *domain.Position
	var operations []domain.Operation

	for i, candle := range candles {
		isLast := i == len(candles)-1

		if position == nil {
			order := strategy.Decide(candle, nil, isLast)
			if order == nil {
				continue
			}
			order.Quantity = balance.Amount() / order.Price.Amount()
			if order.Quantity <= 0 {
				continue
			}
			position = &domain.Position{Buy: *order}
			continue
		}

		order := strategy.Decide(candle, position, isLast)
		if order == nil {
			continue
		}
		order.Quantity = position.Buy.Quantity
		operations = append(operations, position.Close(*order))
		balance = order.Total()
		position = nil
	}

	return operations
}

func TestBuyAndHold_NoCandles(t *testing.T) {
	s := &BuyAndHold{}
	if ops := runStrategy(s, money.Money{}, nil); ops != nil {
		t.Errorf("ops = %+v, want nil", ops)
	}
}

func TestBuyAndHold_BuysLowSellsHigh(t *testing.T) {
	s := &BuyAndHold{}
	candles := []domain.Candle{
		candleWithLowHigh("2010-01-04", 1200, 1242),
		candleWithLowHigh("2010-06-15", 1500, 1550),
		candleWithLowHigh("2010-12-30", 1800, 1899),
	}

	ops := runStrategy(s, newMoney(1200), candles)
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}

	op := ops[0]
	assertOrder(t, "BuyOrder", op.BuyOrder, "2010-01-04", 1200, 1, domain.Buy)
	assertOrder(t, "SellOrder", op.SellOrder, "2010-12-30", 1899, 1, domain.Sell)
	if op.Date != op.BuyOrder.Date {
		t.Errorf("Operation.Date = %q, want %q", op.Date, op.BuyOrder.Date)
	}
}

// TestBuyAndHold_SingleCandleNeverCloses documents a deliberate behavior
// change from the old self-driven implementation, which computed "first"
// and "last" from the whole traversed candle slice up front - so a
// single-candle backtest could buy and sell on the very same day, since
// "first" and "last" were trivially the same candle.
//
// The new Decide-based, streaming model can't replicate that: Decide is
// asked for at most one action per candle (a buy while flat, or an exit
// while holding, never both - see domain.Strategy.Decide), so a position
// opened on a given candle can only be considered for exit starting the
// *next* candle. With only one candle in the whole backtest, there is no
// next candle to ask "should I exit, and is this the last one" on, so the
// position is opened and then dropped, exactly like TwoCandleBreakout drops
// any position still open when candles run out. This is judged an
// acceptable, intentional consequence of the refactor (a single-trading-day
// backtest is a degenerate edge case, not a realistic one) rather than a
// reason to special-case same-candle buy+sell into Backtest's loop, which
// would risk changing TwoCandleBreakout's own carefully-tested "never
// force-close" behavior instead.
func TestBuyAndHold_SingleCandleNeverCloses(t *testing.T) {
	s := &BuyAndHold{}
	candles := []domain.Candle{candleWithLowHigh("2010-01-04", 1200, 1242)}

	if ops := runStrategy(s, newMoney(1200), candles); ops != nil {
		t.Errorf("ops = %+v, want nil (single candle can't be both entry and exit)", ops)
	}
}

func TestBuyAndHold_QuantityScalesWithBalance(t *testing.T) {
	s := &BuyAndHold{}
	candles := []domain.Candle{
		candleWithLowHigh("2010-01-04", 1200, 1242),
		candleWithLowHigh("2010-01-05", 1500, 1600),
	}

	ops := runStrategy(s, newMoney(3600), candles)
	if len(ops) != 1 {
		t.Fatalf("got %d operations, want 1: %+v", len(ops), ops)
	}
	if ops[0].BuyOrder.Quantity != 3 || ops[0].SellOrder.Quantity != 3 {
		t.Errorf("quantity = buy:%d sell:%d, want 3 for both", ops[0].BuyOrder.Quantity, ops[0].SellOrder.Quantity)
	}
}

func TestBuyAndHold_InsufficientBalance(t *testing.T) {
	s := &BuyAndHold{}
	candles := []domain.Candle{candleWithLowHigh("2010-01-04", 1200, 1242)}

	if ops := runStrategy(s, newMoney(500), candles); ops != nil {
		t.Errorf("ops = %+v, want nil (balance can't afford one share)", ops)
	}
}
