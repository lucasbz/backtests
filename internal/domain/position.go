package domain

// Position represents a currently open trade: a buy that hasn't been
// matched with a sell yet. Position is owned by whatever is driving the
// candle traversal (internal/backtest.Backtest), not by the Strategy or
// StopLoss that decided to open/close it.
type Position struct {
	Buy Order
}

// Close pairs this Position's buy with sell, producing the completed
// Operation. Operation.Date mirrors the buy's date, matching how
// Operations were already dated before this type existed.
func (p Position) Close(sell Order) Operation {
	return Operation{Date: p.Buy.Date, BuyOrder: p.Buy, SellOrder: sell}
}
