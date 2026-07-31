package domain

// Strategy decides, one candle at a time, whether to enter or exit a
// position. Backtest owns the candle traversal, the running balance, and
// the currently open Position; it calls Decide once per candle, in
// chronological order.
type Strategy interface {
	Name() string

	// Decide is called once per candle. position is the currently open
	// Position, or nil if flat. isLast is true exactly when candle is the
	// final candle in this backtest's range - it's the only look-ahead
	// information a Strategy gets, needed by strategies like Buy & Hold
	// that guarantee their position closes by the end of the backtest: with
	// no lookahead at all, a purely per-candle Decide can't distinguish
	// "the last candle" from any other one, since candle/position alone
	// don't carry that. Implementations may keep whatever rolling window of
	// recent candles they need internally, but must not track "am I in a
	// position" state themselves - that's owned by Backtest and reflected
	// in the position argument.
	//
	// Returns the order to place this candle - a Buy order if position is
	// nil, a Sell order if it isn't - or nil if no action should be taken
	// this candle.
	//
	// Buy orders returned here should leave Quantity 0: Backtest fills it
	// in from the running balance and the order's Price. Sell orders should
	// set Quantity to position.Buy.Quantity.
	Decide(candle Candle, position *Position, isLast bool) *Order
}
