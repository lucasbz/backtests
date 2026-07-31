package domain

import "github.com/Rhymond/go-money"

type Operation struct {
	Date      string `json:"date"`
	BuyOrder  Order  `json:"buyOrder"`
	SellOrder Order  `json:"sellOrder"`
}

// OperationOutcome classifies whether an Operation closed at a profit or a
// loss. Breaking even (zero profit) counts as a Gain, to keep this binary.
type OperationOutcome string

const (
	Gain OperationOutcome = "Gain"
	Loss OperationOutcome = "Loss"
)

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

// Outcome classifies this operation's Profit as a Gain or a Loss.
func (o Operation) Outcome() (OperationOutcome, error) {
	profit, err := o.Profit()
	if err != nil {
		return "", err
	}
	if profit.IsNegative() {
		return Loss, nil
	}
	return Gain, nil
}
