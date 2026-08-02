package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Rhymond/go-money"
)

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

// Days is the whole number of calendar days between the buy and sell
// orders' dates - how long this operation was open.
func (o Operation) Days() (int, error) {
	buyDate, err := time.Parse("2006-01-02", o.BuyOrder.Date)
	if err != nil {
		return 0, fmt.Errorf("parsing buy date %s: %w", o.BuyOrder.Date, err)
	}
	sellDate, err := time.Parse("2006-01-02", o.SellOrder.Date)
	if err != nil {
		return 0, fmt.Errorf("parsing sell date %s: %w", o.SellOrder.Date, err)
	}
	return int(sellDate.Sub(buyDate).Hours() / 24), nil
}

// operationJSON mirrors Operation but with a computed Days field added, same
// shadow-struct pattern as backtest.BacktestResult's backtestResultJSON.
type operationJSON struct {
	Date      string `json:"date"`
	BuyOrder  Order  `json:"buyOrder"`
	SellOrder Order  `json:"sellOrder"`
	Days      int    `json:"days"`
}

func (o Operation) MarshalJSON() ([]byte, error) {
	days, err := o.Days()
	if err != nil {
		return nil, err
	}
	return json.Marshal(operationJSON{
		Date:      o.Date,
		BuyOrder:  o.BuyOrder,
		SellOrder: o.SellOrder,
		Days:      days,
	})
}
