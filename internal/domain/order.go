package domain

import "github.com/Rhymond/go-money"

const (
	Sell OrderType = "Sell"
	Buy  OrderType = "Buy"
)

type OrderType string

type Order struct {
	Date      string
	Price     money.Money
	Quantity  int64
	OrderType OrderType
}

// Total is the order's price times quantity.
func (o Order) Total() money.Money {
	return *o.Price.Multiply(o.Quantity)
}
