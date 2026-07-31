package domain

import (
	"encoding/json"

	"github.com/Rhymond/go-money"
)

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

// orderJSON mirrors the plain-decimal shape candleJSON uses: Price is a bare
// JSON number, not go-money's default {"amount":...,"currency":...} object
// form. OrderType is deliberately omitted: callers marshal Order only as a
// BuyOrder/SellOrder field of Operation, where the key already says which it
// is.
type orderJSON struct {
	Date     string  `json:"date"`
	Price    float64 `json:"price"`
	Quantity int64   `json:"quantity"`
}

func (o Order) MarshalJSON() ([]byte, error) {
	return json.Marshal(orderJSON{
		Date:     o.Date,
		Price:    o.Price.AsMajorUnits(),
		Quantity: o.Quantity,
	})
}
