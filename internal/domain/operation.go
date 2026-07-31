package domain

import "github.com/Rhymond/go-money"

type Operation struct {
	Date      string
	BuyOrder  Order
	SellOrder Order
}

// Profit is this operation's gain or loss: what the sell order returned
// minus what the buy order cost.
func (o Operation) Profit() (money.Money, error) {
	buyTotal := o.BuyOrder.Total()
	sellTotal := o.SellOrder.Total()

	profit, err := sellTotal.Subtract(&buyTotal)
	if err != nil {
		return money.Money{}, err
	}
	return *profit, nil
}
