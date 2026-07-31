package domain

// StopLoss is a risk-control rule that races a Strategy's own exit signal:
// once a position is open, whichever fires first on a given candle - the
// strategy's own Decide, or the stop-loss's Check - closes the position.
// StopLoss is only ever consulted while a position is open; it has no role
// in deciding when to enter one.
type StopLoss interface {
	Name() string

	// Check is called once per candle while a position is open. Returns the
	// sell order to place if the stop-loss condition is triggered this
	// candle, or nil otherwise. Callers fill in Quantity from the position.
	Check(candle Candle, position Position) *Order
}
